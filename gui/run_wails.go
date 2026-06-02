//go:build wails

package gui

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func Run() error {
	app := NewApp()
	return wails.Run(&options.App{
		Title:            "Beam",
		Width:            1200,
		Height:           800,
		MinWidth:         900,
		MinHeight:        600,
		BackgroundColour: &options.RGBA{R: 17, G: 24, B: 39, A: 255},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup: app.startup,
		Bind:      []interface{}{app},
	})
}
