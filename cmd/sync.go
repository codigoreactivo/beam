package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var syncDryRun bool

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync local directory → remote (incremental)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if flagProject == "" {
			return fmt.Errorf("--project / -p is required")
		}
		if syncDryRun {
			fmt.Fprintf(cmd.OutOrStdout(), "dry-run sync for %q — not yet implemented\n", flagProject)
			return nil
		}
		fmt.Fprintf(cmd.OutOrStdout(), "syncing project %q — not yet implemented\n", flagProject)
		return nil
	},
}

func init() {
	syncCmd.Flags().BoolVar(&syncDryRun, "dry-run", false, "preview what would be synced without transferring")
	rootCmd.AddCommand(syncCmd)
}
