package transfer

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jesusjhoel/beam/internal/config"
)

type Entry struct {
	Name    string      `json:"name"`
	Size    int64       `json:"size"`
	IsDir   bool        `json:"is_dir"`
	Mode    os.FileMode `json:"mode"`
	ModTime time.Time   `json:"mod_time"`
}

type WalkEntry struct {
	RelPath string // relative to the walk root, always forward slashes
	Entry   Entry
}

type Client interface {
	Upload(ctx context.Context, local, remote string) (int64, error)
	Download(ctx context.Context, remote, local string) (int64, error)
	List(ctx context.Context, path string) ([]Entry, error)
	Walk(ctx context.Context, path string) ([]WalkEntry, error)
	Remove(ctx context.Context, path string) error
	Rename(ctx context.Context, src, dst string) error
	Mkdir(ctx context.Context, path string) error
	Close() error
}

func Connect(p *config.Project) (Client, error) {
	switch p.Protocol {
	case config.ProtocolSFTP:
		return dialSFTP(p)
	case config.ProtocolFTP, config.ProtocolFTPS:
		return nil, fmt.Errorf("protocol %q not yet implemented — coming soon", p.Protocol)
	default:
		return nil, fmt.Errorf("unknown protocol %q", p.Protocol)
	}
}
