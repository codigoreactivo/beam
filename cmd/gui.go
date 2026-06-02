package cmd

import (
	"github.com/codigoreactivo/beam/gui"
	"github.com/spf13/cobra"
)

var guiCmd = &cobra.Command{
	Use:   "gui",
	Short: "Launch desktop GUI (Wails)",
	Long: `Launch the Beam desktop GUI built with Wails + React.

Requires a Wails build — install Wails then run:
  wails build -tags wails

Or for live development:
  wails dev`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return gui.Run()
	},
}

func init() {
	rootCmd.AddCommand(guiCmd)
}
