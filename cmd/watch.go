package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var watchOnce bool

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch local directory and auto-sync on change",
	RunE: func(cmd *cobra.Command, args []string) error {
		if flagProject == "" {
			return fmt.Errorf("--project / -p is required")
		}
		if watchOnce {
			fmt.Fprintf(cmd.OutOrStdout(), "sync-once for %q — not yet implemented\n", flagProject)
			return nil
		}
		fmt.Fprintf(cmd.OutOrStdout(), "watching project %q — not yet implemented\n", flagProject)
		return nil
	},
}

func init() {
	watchCmd.Flags().BoolVar(&watchOnce, "once", false, "sync once then exit")
	rootCmd.AddCommand(watchCmd)
}
