package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Show diff between local and remote directories",
	RunE: func(cmd *cobra.Command, args []string) error {
		if flagProject == "" {
			return fmt.Errorf("--project / -p is required")
		}
		fmt.Fprintf(cmd.OutOrStdout(), "diff for %q — not yet implemented\n", flagProject)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(diffCmd)
}
