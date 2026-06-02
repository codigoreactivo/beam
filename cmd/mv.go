package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var mvCmd = &cobra.Command{
	Use:   "mv <src> <dst>",
	Short: "Rename or move a remote file",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if flagProject == "" {
			return fmt.Errorf("--project / -p is required")
		}
		fmt.Fprintf(cmd.OutOrStdout(), "mv %q → %q on %q — not yet implemented\n", args[0], args[1], flagProject)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(mvCmd)
}
