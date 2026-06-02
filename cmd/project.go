package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/tabwriter"

	"github.com/jesusjhoel/beam/internal/config"
	"github.com/jesusjhoel/beam/internal/project"
	"github.com/spf13/cobra"
)

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage connection profiles",
}

// ── list ──────────────────────────────────────────────────────────────────────

var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all projects",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := loadManager()
		if err != nil {
			return err
		}
		projects := mgr.All()

		if flagJSON {
			return printJSON(cmd, projects)
		}

		if len(projects) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No projects yet. Run: beam project add <name>")
			return nil
		}

		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tPROTOCOL\tHOST\tENV")
		for _, p := range projects {
			host := fmt.Sprintf("%s:%d", p.Host, p.Port)
			env := p.Env
			if env == "" {
				env = "—"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", p.Name, p.Protocol, host, env)
		}
		return w.Flush()
	},
}

// ── add ───────────────────────────────────────────────────────────────────────

var (
	addHost   string
	addUser   string
	addProto  string
	addLocal  string
	addRemote string
	addPort   int
	addKey    string
	addEnv    string
)

var projectAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a project (interactive wizard or --flags)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		mgr, err := loadManager()
		if err != nil {
			return err
		}

		p := config.Project{Name: name}

		if isInteractive() {
			p, err = wizard(name)
			if err != nil {
				return err
			}
		} else {
			if addHost == "" {
				return fmt.Errorf("--host is required (or run without flags for interactive wizard)")
			}
			p.Protocol = config.Protocol(addProto)
			p.Host = addHost
			p.User = addUser
			p.Port = addPort
			p.Key = addKey
			p.Local = addLocal
			p.Remote = addRemote
			p.Env = addEnv
			if p.Port == 0 {
				p.Port = p.DefaultPort()
			}
		}

		if err := mgr.Add(p); err != nil {
			return err
		}

		if !flagQuiet {
			fmt.Fprintf(cmd.OutOrStdout(), "✓ Project %q added (%s@%s:%d)\n",
				p.Name, p.User, p.Host, p.Port)
		}
		return nil
	},
}

// ── remove ────────────────────────────────────────────────────────────────────

var projectRemoveCmd = &cobra.Command{
	Use:     "remove <name>",
	Aliases: []string{"rm", "delete"},
	Short:   "Remove a project",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := loadManager()
		if err != nil {
			return err
		}
		if err := mgr.Remove(args[0]); err != nil {
			return err
		}
		if !flagQuiet {
			fmt.Fprintf(cmd.OutOrStdout(), "✓ Project %q removed\n", args[0])
		}
		return nil
	},
}

// ── edit ──────────────────────────────────────────────────────────────────────

var projectEditCmd = &cobra.Command{
	Use:   "edit <name>",
	Short: "Open projects.yaml in $EDITOR",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vi"
		}
		path := config.BeamDir() + "/projects.yaml"
		c := exec.Command(editor, path)
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	},
}

// ── show ──────────────────────────────────────────────────────────────────────

var projectShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show project details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := loadManager()
		if err != nil {
			return err
		}
		p, err := mgr.Get(args[0])
		if err != nil {
			return err
		}

		if flagJSON {
			return printJSON(cmd, p)
		}

		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "Name:\t%s\n", p.Name)
		fmt.Fprintf(w, "Protocol:\t%s\n", p.Protocol)
		fmt.Fprintf(w, "Host:\t%s:%d\n", p.Host, p.Port)
		fmt.Fprintf(w, "User:\t%s\n", p.User)
		if p.Key != "" {
			fmt.Fprintf(w, "Key:\t%s\n", p.Key)
		}
		if p.Password != "" {
			fmt.Fprintf(w, "Password:\t(set)\n")
		}
		fmt.Fprintf(w, "Local:\t%s\n", p.Local)
		fmt.Fprintf(w, "Remote:\t%s\n", p.Remote)
		if p.Env != "" {
			fmt.Fprintf(w, "Env:\t%s\n", p.Env)
		}
		return w.Flush()
	},
}

// ── test ──────────────────────────────────────────────────────────────────────

var projectTestCmd = &cobra.Command{
	Use:   "test <name>",
	Short: "Validate project config (connection test coming soon)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := loadManager()
		if err != nil {
			return err
		}
		p, err := mgr.Get(args[0])
		if err != nil {
			return err
		}

		var issues []string
		if p.Host == "" {
			issues = append(issues, "host is empty")
		}
		if p.User == "" {
			issues = append(issues, "user is empty")
		}
		if p.Local == "" {
			issues = append(issues, "local directory is empty")
		}
		if p.Remote == "" {
			issues = append(issues, "remote directory is empty")
		}
		if p.Protocol == config.ProtocolSFTP && p.Key == "" && p.Password == "" {
			issues = append(issues, "sftp requires key or password")
		}

		if len(issues) > 0 {
			for _, iss := range issues {
				fmt.Fprintf(cmd.OutOrStdout(), "✗ %s\n", iss)
			}
			return fmt.Errorf("project %q has configuration errors", args[0])
		}

		fmt.Fprintf(cmd.OutOrStdout(), "✓ Config for %q looks valid (%s %s@%s:%d)\n",
			p.Name, p.Protocol, p.User, p.Host, p.Port)
		fmt.Fprintln(cmd.OutOrStdout(), "  Connection test will be available once the transfer engine is implemented.")
		return nil
	},
}

// ── helpers ───────────────────────────────────────────────────────────────────

func loadManager() (*project.Manager, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	return project.NewManager(cfg)
}

func printJSON(cmd *cobra.Command, v any) error {
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func isInteractive() bool {
	return addHost == "" && addUser == "" && addLocal == "" && addRemote == ""
}

func prompt(reader *bufio.Reader, label, def string) string {
	if def != "" {
		fmt.Printf("%s [%s]: ", label, def)
	} else {
		fmt.Printf("%s: ", label)
	}
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return def
	}
	return line
}

func wizard(name string) (config.Project, error) {
	r := bufio.NewReader(os.Stdin)
	p := config.Project{Name: name}

	fmt.Printf("\n  Adding project %q\n\n", name)

	proto := prompt(r, "Protocol (sftp/ftp/ftps)", "sftp")
	p.Protocol = config.Protocol(proto)

	p.Host = prompt(r, "Host", "")
	if p.Host == "" {
		return p, fmt.Errorf("host is required")
	}

	defaultPort := p.DefaultPort()
	portStr := prompt(r, fmt.Sprintf("Port"), fmt.Sprintf("%d", defaultPort))
	port := defaultPort
	fmt.Sscan(portStr, &port)
	p.Port = port

	p.User = prompt(r, "User", "")
	if p.User == "" {
		return p, fmt.Errorf("user is required")
	}

	if p.Protocol == config.ProtocolSFTP {
		auth := prompt(r, "Auth (password/key)", "key")
		if auth == "key" {
			p.Key = prompt(r, "Key path", "~/.ssh/id_rsa")
		} else {
			p.Password = prompt(r, "Password", "")
		}
	} else {
		p.Password = prompt(r, "Password", "")
	}

	p.Local = prompt(r, "Local directory", "./")
	p.Remote = prompt(r, "Remote directory", "/")
	p.Env = prompt(r, "Environment (dev/staging/production)", "")
	if p.Port == 0 {
		p.Port = p.DefaultPort()
	}

	fmt.Println()
	return p, nil
}

func init() {
	projectAddCmd.Flags().StringVar(&addHost, "host", "", "remote host")
	projectAddCmd.Flags().StringVar(&addUser, "user", "", "remote user")
	projectAddCmd.Flags().StringVar(&addProto, "proto", "sftp", "protocol (sftp|ftp|ftps)")
	projectAddCmd.Flags().StringVar(&addLocal, "local", "", "local directory")
	projectAddCmd.Flags().StringVar(&addRemote, "remote", "", "remote directory")
	projectAddCmd.Flags().IntVar(&addPort, "port", 0, "remote port (default: protocol default)")
	projectAddCmd.Flags().StringVar(&addKey, "key", "", "path to SSH private key (sftp)")
	projectAddCmd.Flags().StringVar(&addEnv, "env", "", "environment label (dev/staging/production)")

	projectCmd.AddCommand(
		projectListCmd,
		projectAddCmd,
		projectRemoveCmd,
		projectEditCmd,
		projectShowCmd,
		projectTestCmd,
	)
	rootCmd.AddCommand(projectCmd)
}
