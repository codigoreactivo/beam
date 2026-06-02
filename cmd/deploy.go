package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Full deploy: sync local → remote then run post-deploy hooks",
	RunE: func(cmd *cobra.Command, args []string) error {
		if flagProject == "" {
			return fmt.Errorf("--project / -p is required")
		}
		fmt.Fprintf(cmd.OutOrStdout(), "deploying project %q — not yet implemented\n", flagProject)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(deployCmd)
}
