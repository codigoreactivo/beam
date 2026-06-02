package watcher

import (
	"context"
	"fmt"

	"github.com/jesusjhoel/beam/internal/config"
)

type ChangeEvent struct {
	Path    string
	Project *config.Project
}

type Watcher struct {
	project *config.Project
	events  chan ChangeEvent
}

func New(project *config.Project) *Watcher {
	return &Watcher{
		project: project,
		events:  make(chan ChangeEvent, 64),
	}
}

func (w *Watcher) Events() <-chan ChangeEvent {
	return w.events
}

func (w *Watcher) Watch(ctx context.Context) error {
	_ = ctx
	return fmt.Errorf("not implemented")
}

func (w *Watcher) Close() error {
	close(w.events)
	return nil
}
