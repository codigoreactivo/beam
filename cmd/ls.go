package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var lsCmd = &cobra.Command{
	Use:   "ls [remote-path]",
	Short: "List remote directory",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if flagProject == "" {
			return fmt.Errorf("--project / -p is required")
		}
		path := "/"
		if len(args) > 0 {
			path = args[0]
		}
		fmt.Fprintf(cmd.OutOrStdout(), "ls %q on %q — not yet implemented\n", path, flagProject)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(lsCmd)
}
