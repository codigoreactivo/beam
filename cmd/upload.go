package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var uploadCmd = &cobra.Command{
	Use:   "upload <local> [remote]",
	Short: "Upload a file or directory to remote",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if flagProject == "" {
			return fmt.Errorf("--project / -p is required")
		}
		local := args[0]
		remote := ""
		if len(args) > 1 {
			remote = args[1]
		}
		fmt.Fprintf(cmd.OutOrStdout(), "upload %q → %q on %q — not yet implemented\n", local, remote, flagProject)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(uploadCmd)
}
