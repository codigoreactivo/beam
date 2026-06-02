package cmd

import (
	"github.com/codigoreactivo/beam/tui"
	"github.com/spf13/cobra"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch interactive TUI dashboard",
	RunE: func(cmd *cobra.Command, args []string) error {
		return tui.Run()
	},
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}
