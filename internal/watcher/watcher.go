package watcher

import (
	"context"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/codigoreactivo/beam/internal/config"
	"github.com/codigoreactivo/beam/internal/ignore"
)

const debounceDelay = 500 * time.Millisecond

type ChangeEvent struct {
	Path    string
	RelPath string
	Project *config.Project
}

type Watcher struct {
	project *config.Project
	rules   *ignore.Rules
	fsw     *fsnotify.Watcher
	Events  chan ChangeEvent
	mu      sync.Mutex
	timers  map[string]*time.Timer
}

func New(p *config.Project) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &Watcher{
		project: p,
		rules:   ignore.Load(p.Local),
		fsw:     fsw,
		Events:  make(chan ChangeEvent, 64),
		timers:  map[string]*time.Timer{},
	}, nil
}

// Start adds all directories under the project local root to the watch list
// and begins the event loop in a goroutine. Returns an error if the local
// directory cannot be traversed.
func (w *Watcher) Start(ctx context.Context) error {
	err := filepath.WalkDir(w.project.Local, func(absPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			rel, _ := filepath.Rel(w.project.Local, absPath)
			relSlash := filepath.ToSlash(rel)
			if shouldIgnoreDir(d.Name()) || w.rules.Match(relSlash, true) {
				return filepath.SkipDir
			}
			return w.fsw.Add(absPath)
		}
		return nil
	})
	if err != nil {
		return err
	}
	go w.loop(ctx)
	return nil
}

func (w *Watcher) Close() error {
	return w.fsw.Close()
}

func (w *Watcher) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-w.fsw.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
				rel, _ := filepath.Rel(w.project.Local, event.Name)
				relSlash := filepath.ToSlash(rel)
				if !shouldIgnorePath(event.Name) && !w.rules.Match(relSlash, false) {
					w.debounce(event.Name)
				}
			}
		case <-w.fsw.Errors:
			// non-fatal: log and continue
		}
	}
}

func (w *Watcher) debounce(path string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if t, ok := w.timers[path]; ok {
		t.Reset(debounceDelay)
		return
	}
	w.timers[path] = time.AfterFunc(debounceDelay, func() {
		w.mu.Lock()
		delete(w.timers, path)
		w.mu.Unlock()

		rel, _ := filepath.Rel(w.project.Local, path)
		w.Events <- ChangeEvent{
			Path:    path,
			RelPath: filepath.ToSlash(rel),
			Project: w.project,
		}
	})
}

var ignoreDirs = map[string]bool{
	".git": true, "node_modules": true, ".svn": true, ".hg": true,
}

var ignoreSuffixes = []string{".tmp", ".log", ".swp", "~"}

func shouldIgnoreDir(name string) bool {
	return ignoreDirs[name]
}

func shouldIgnorePath(path string) bool {
	base := filepath.Base(path)
	for _, s := range ignoreSuffixes {
		if strings.HasSuffix(base, s) {
			return true
		}
	}
	return false
}
