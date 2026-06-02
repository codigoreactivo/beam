package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	flagProject string
	flagConfig  string
	flagQuiet   bool
	flagJSON    bool
)

var rootCmd = &cobra.Command{
	Use:   "beam",
	Short: "Modern FTP/SFTP client with CLI, TUI, GUI and MCP interfaces",
	Long: `Beam is a modern FTP/SFTP client for automating web deployment workflows.
Supports multiple projects concurrently with file watching, hooks, and AI automation.`,
	SilenceErrors: true,
	SilenceUsage:  true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&flagProject, "project", "p", "", "target project name")
	rootCmd.PersistentFlags().StringVar(&flagConfig, "config", "", "config file (default: ~/.beam/beam.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&flagQuiet, "quiet", "q", false, "suppress output except errors")
	rootCmd.PersistentFlags().BoolVar(&flagJSON, "json", false, "output as JSON")
}
