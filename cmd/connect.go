package cmd

import (
	"fmt"

	"github.com/codigoreactivo/beam/internal/config"
	"github.com/codigoreactivo/beam/internal/transfer"
)

// dial resolves the named project and opens a transfer client.
// Caller is responsible for calling client.Close().
func dial(name string) (*config.Project, transfer.Client, error) {
	if name == "" {
		return nil, nil, fmt.Errorf("--project / -p is required")
	}
	mgr, err := loadManager()
	if err != nil {
		return nil, nil, err
	}
	p, err := mgr.Get(name)
	if err != nil {
		return nil, nil, err
	}
	client, err := transfer.Connect(p)
	if err != nil {
		return nil, nil, fmt.Errorf("connect to %q: %w", p.Name, err)
	}
	return p, client, nil
}
