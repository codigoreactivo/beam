package sync

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/codigoreactivo/beam/internal/config"
	"github.com/codigoreactivo/beam/internal/ignore"
	"github.com/codigoreactivo/beam/internal/transfer"
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
	rules := ignore.Load(p.Local)
	localMap, err := walkLocal(p.Local, rules)
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

const maxRetries = 3

// Execute uploads every non-skipped entry in the plan using a worker pool.
// onFile is called before each upload attempt (can be nil).
func Execute(ctx context.Context, client transfer.Client, plan *Plan, p *config.Project, onFile func(PlanEntry)) error {
	workers := p.Workers
	if workers <= 0 {
		workers = 4
	}
	// FTP/FTPS share a single control connection — must be serial
	if p.Protocol == config.ProtocolFTP || p.Protocol == config.ProtocolFTPS {
		workers = 1
	}

	type result struct {
		relPath string
		err     error
	}

	sem := make(chan struct{}, workers)
	errc := make(chan result, len(plan.Entries))
	var wg sync.WaitGroup

	for _, entry := range plan.Entries {
		if entry.Action == ActionSkip {
			continue
		}
		if ctx.Err() != nil {
			break
		}
		entry := entry
		sem <- struct{}{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			if onFile != nil {
				onFile(entry)
			}
			local := filepath.Join(p.Local, filepath.FromSlash(entry.RelPath))
			remote := path.Join(p.Remote, entry.RelPath)
			err := uploadWithRetry(ctx, client, local, remote, maxRetries)
			errc <- result{relPath: entry.RelPath, err: err}
		}()
	}

	wg.Wait()
	close(errc)

	for r := range errc {
		if r.err != nil {
			return fmt.Errorf("upload %s: %w", r.relPath, r.err)
		}
	}
	return ctx.Err()
}

// uploadWithRetry retries on transient errors with exponential backoff.
func uploadWithRetry(ctx context.Context, client transfer.Client, local, remote string, attempts int) error {
	var err error
	for i := range attempts {
		if i > 0 {
			delay := time.Duration(i*i) * 500 * time.Millisecond
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}
		if _, err = client.Upload(ctx, local, remote); err == nil {
			return nil
		}
	}
	return err
}

// ── internals ────────────────────────────────────────────────────────────────

func walkLocal(root string, rules *ignore.Rules) (map[string]os.FileInfo, error) {
	files := map[string]os.FileInfo{}
	err := filepath.Walk(root, func(localPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, localPath)
		relSlash := filepath.ToSlash(rel)
		if info.IsDir() {
			if shouldIgnoreDir(info.Name()) || rules.Match(relSlash, true) {
				return filepath.SkipDir
			}
			return nil
		}
		if shouldIgnoreFile(info.Name()) || rules.Match(relSlash, false) {
			return nil
		}
		files[relSlash] = info
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
