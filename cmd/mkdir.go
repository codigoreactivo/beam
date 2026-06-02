package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var mkdirCmd = &cobra.Command{
	Use:   "mkdir <remote-path>",
	Short: "Create a remote directory",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if flagProject == "" {
			return fmt.Errorf("--project / -p is required")
		}
		fmt.Fprintf(cmd.OutOrStdout(), "mkdir %q on %q — not yet implemented\n", args[0], flagProject)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(mkdirCmd)
}
