//go:build !wails

package gui

import "errors"

// Run returns an error — GUI requires a Wails build.
// Install Wails: go install github.com/wailsapp/wails/v2/cmd/wails@latest
// Build:         wails build -tags wails
func Run() error {
	return errors.New("GUI requires a Wails build: wails build -tags wails")
}
