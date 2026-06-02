package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	logsAll    bool
	logsExport bool
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Tail project logs",
	RunE: func(cmd *cobra.Command, args []string) error {
		if logsAll {
			fmt.Fprintln(cmd.OutOrStdout(), "logs --all — not yet implemented")
			return nil
		}
		if flagProject == "" {
			return fmt.Errorf("--project / -p is required (or use --all)")
		}
		if logsExport {
			fmt.Fprintf(cmd.OutOrStdout(), "exporting logs for %q — not yet implemented\n", flagProject)
			return nil
		}
		fmt.Fprintf(cmd.OutOrStdout(), "tailing logs for %q — not yet implemented\n", flagProject)
		return nil
	},
}

func init() {
	logsCmd.Flags().BoolVar(&logsAll, "all", false, "tail logs for all projects")
	logsCmd.Flags().BoolVar(&logsExport, "export", false, "export logs to file")
	rootCmd.AddCommand(logsCmd)
}
