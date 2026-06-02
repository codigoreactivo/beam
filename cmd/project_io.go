package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/codigoreactivo/beam/internal/config"
	"github.com/codigoreactivo/beam/internal/exporter"
	"github.com/codigoreactivo/beam/internal/importer"
	"github.com/spf13/cobra"
)

// ── import ────────────────────────────────────────────────────────────────────

var (
	importLocal    string
	importOverride bool
)

var projectImportCmd = &cobra.Command{
	Use:   "import <file>",
	Short: "Import projects from a config file",
	Long: `Import FTP/SFTP projects from a configuration file.

Supported formats:
  Beam JSON    beam-backup.json
  FileZilla    sitemanager.xml
  Cyberduck    profile.cyberduckprofile  (cPanel "Configuration Files")
  CoreFTP      sites.xml                 (cPanel "Configuration Files")
  .env         .env  (FTP_HOST, FTP_USER, FTP_PASS, FTP_REMOTE)

Examples:
  beam project import ~/Downloads/profile.cyberduckprofile
  beam project import ~/Downloads/sitemanager.xml --local ./my-site
  beam project import backup.json`,
	Args: cobra.ExactArgs(1),
	RunE: runImport,
}

func runImport(cmd *cobra.Command, args []string) error {
	projects, err := importer.Parse(args[0])
	if err != nil {
		return err
	}

	mgr, err := loadManager()
	if err != nil {
		return err
	}

	added, skipped := 0, 0
	var noLocal bool
	for _, p := range projects {
		if importLocal != "" {
			p.Local = importLocal
		}
		if p.Port == 0 {
			p.Port = p.DefaultPort()
		}
		if p.Local == "" {
			noLocal = true
		}

		if err := mgr.Add(p); err != nil {
			if importOverride {
				_ = mgr.Remove(p.Name)
				_ = mgr.Add(p)
				fmt.Fprintf(cmd.OutOrStdout(), "  ~ updated  %s (%s@%s:%d)\n", p.Name, p.User, p.Host, p.Port)
				added++
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "  ! skipped  %s — already exists (--override to replace)\n", p.Name)
				skipped++
			}
			continue
		}
		fmt.Fprintf(cmd.OutOrStdout(), "  + imported %s (%s@%s:%d)\n", p.Name, p.User, p.Host, p.Port)
		added++
	}

	if !flagQuiet {
		extra := ""
		if noLocal {
			extra = "\n⚠  Some projects have no local path — set it with: beam project edit"
		}
		fmt.Fprintf(cmd.OutOrStdout(), "\n✓ %d imported, %d skipped from %s%s\n",
			added, skipped, filepath.Base(args[0]), extra)
	}
	return nil
}

// ── export ────────────────────────────────────────────────────────────────────

var (
	exportFormat string
	exportOutput string
)

var projectExportCmd = &cobra.Command{
	Use:   "export [name]",
	Short: "Export projects to a config file",
	Long: `Export one or all projects to a config file.

Formats:
  beam       Beam JSON backup (default) — portable, re-importable
  filezilla  FileZilla sitemanager.xml
  env        .env file (FTP_HOST=, FTP_USER=, ...) — single project only

Examples:
  beam project export                             # all projects → stdout
  beam project export mysite                      # one project  → stdout
  beam project export --format filezilla -o sites.xml
  beam project export --format env mysite         # .env to stdout
  beam project export -o backup.json              # all projects → file`,
	Args: cobra.MaximumNArgs(1),
	RunE: runExport,
}

func runExport(cmd *cobra.Command, args []string) error {
	mgr, err := loadManager()
	if err != nil {
		return err
	}

	var list []config.Project
	if len(args) == 1 {
		p, err := mgr.Get(args[0])
		if err != nil {
			return err
		}
		list = []config.Project{*p}
	} else {
		list = mgr.All()
	}

	if len(list) == 0 {
		return fmt.Errorf("no projects to export")
	}

	if exportFormat == exporter.FormatEnv && len(list) > 1 {
		return fmt.Errorf("--format env supports one project at a time; add a name: beam project export --format env <name>")
	}

	data, err := exporter.Marshal(list, exportFormat)
	if err != nil {
		return err
	}

	if exportOutput != "" {
		if filepath.Ext(exportOutput) == "" {
			exportOutput += exporter.Ext(exportFormat)
		}
		if err := os.WriteFile(exportOutput, data, 0o600); err != nil {
			return fmt.Errorf("write %s: %w", exportOutput, err)
		}
		if !flagQuiet {
			label := "all projects"
			if len(args) == 1 {
				label = fmt.Sprintf("%q", args[0])
			}
			fmt.Fprintf(cmd.OutOrStdout(), "✓ exported %s → %s\n", label, exportOutput)
		}
		return nil
	}

	_, err = cmd.OutOrStdout().Write(data)
	return err
}

func init() {
	projectImportCmd.Flags().StringVar(&importLocal, "local", "", "set local directory for all imported projects")
	projectImportCmd.Flags().BoolVar(&importOverride, "override", false, "replace existing projects with same name")

	projectExportCmd.Flags().StringVar(&exportFormat, "format", exporter.FormatBeam, "output format: beam | filezilla | env")
	projectExportCmd.Flags().StringVarP(&exportOutput, "output", "o", "", "write to file (default: stdout)")

	projectCmd.AddCommand(projectImportCmd, projectExportCmd)
}
