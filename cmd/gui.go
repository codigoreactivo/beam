package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var guiCmd = &cobra.Command{
	Use:   "gui",
	Short: "Launch desktop GUI (Wails)",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintln(cmd.OutOrStdout(), "GUI — not yet implemented")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(guiCmd)
}
