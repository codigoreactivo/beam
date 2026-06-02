package sync

import (
	"context"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/jesusjhoel/beam/internal/config"
	"github.com/jesusjhoel/beam/internal/transfer"
)

type Action string

const (
	ActionAdd    Action = "add"
	ActionUpdate Action = "update"
	ActionSkip   Action = "skip"
)

type PlanEntry struct {
	RelPath     string
	Action      Action
	LocalSize   int64
	RemoteSize  int64
	LocalMtime  time.Time
	RemoteMtime time.Time
}

type Plan struct {
	Entries     []PlanEntry
	AddCount    int
	UpdateCount int
	SkipCount   int
}

func (p *Plan) TotalBytes() int64 {
	var n int64
	for _, e := range p.Entries {
		if e.Action != ActionSkip {
			n += e.LocalSize
		}
	}
	return n
}

// Build walks the local directory and the remote directory, then
// produces a plan of what needs to be uploaded.
func Build(ctx context.Context, client transfer.Client, p *config.Project) (*Plan, error) {
	localMap, err := walkLocal(p.Local)
	if err != nil {
		return nil, err
	}

	remoteEntries, err := client.Walk(ctx, p.Remote)
	if err != nil {
		return nil, err
	}
	remoteMap := make(map[string]transfer.Entry, len(remoteEntries))
	for _, e := range remoteEntries {
		if !e.Entry.IsDir {
			remoteMap[e.RelPath] = e.Entry
		}
	}

	plan := &Plan{}
	for relPath, localInfo := range localMap {
		remote, exists := remoteMap[relPath]
		action := decide(localInfo, remote, exists)
		plan.Entries = append(plan.Entries, PlanEntry{
			RelPath:     relPath,
			Action:      action,
			LocalSize:   localInfo.Size(),
			RemoteSize:  remote.Size,
			LocalMtime:  localInfo.ModTime(),
			RemoteMtime: remote.ModTime,
		})
		switch action {
		case ActionAdd:
			plan.AddCount++
		case ActionUpdate:
			plan.UpdateCount++
		case ActionSkip:
			plan.SkipCount++
		}
	}
	return plan, nil
}

// Execute uploads every non-skipped entry in the plan.
// onFile is called before each upload (can be nil).
func Execute(ctx context.Context, client transfer.Client, plan *Plan, p *config.Project, onFile func(PlanEntry)) error {
	for _, entry := range plan.Entries {
		if entry.Action == ActionSkip {
			continue
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if onFile != nil {
			onFile(entry)
		}
		local := filepath.Join(p.Local, filepath.FromSlash(entry.RelPath))
		remote := path.Join(p.Remote, entry.RelPath)
		if _, err := client.Upload(ctx, local, remote); err != nil {
			return err
		}
	}
	return nil
}

// ── internals ────────────────────────────────────────────────────────────────

func walkLocal(root string) (map[string]os.FileInfo, error) {
	files := map[string]os.FileInfo{}
	err := filepath.Walk(root, func(localPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if shouldIgnoreDir(info.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if shouldIgnoreFile(info.Name()) {
			return nil
		}
		rel, _ := filepath.Rel(root, localPath)
		files[filepath.ToSlash(rel)] = info
		return nil
	})
	return files, err
}

func decide(local os.FileInfo, remote transfer.Entry, exists bool) Action {
	if !exists {
		return ActionAdd
	}
	if local.Size() != remote.Size {
		return ActionUpdate
	}
	// 1-second tolerance: SFTP timestamps can lose sub-second precision
	if local.ModTime().After(remote.ModTime.Add(time.Second)) {
		return ActionUpdate
	}
	return ActionSkip
}

var ignoreDirs = map[string]bool{
	".git": true, "node_modules": true, ".svn": true, ".hg": true,
}

var ignoreSuffixes = []string{".tmp", ".log", ".DS_Store", "Thumbs.db"}

func shouldIgnoreDir(name string) bool {
	return ignoreDirs[name]
}

func shouldIgnoreFile(name string) bool {
	for _, s := range ignoreSuffixes {
		if strings.HasSuffix(name, s) || name == s {
			return true
		}
	}
	return false
}
