package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var downloadCmd = &cobra.Command{
	Use:   "download <remote> [local]",
	Short: "Download a file or directory from remote",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if flagProject == "" {
			return fmt.Errorf("--project / -p is required")
		}
		remote := args[0]
		local := ""
		if len(args) > 1 {
			local = args[1]
		}
		fmt.Fprintf(cmd.OutOrStdout(), "download %q → %q from %q — not yet implemented\n", remote, local, flagProject)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(downloadCmd)
}
