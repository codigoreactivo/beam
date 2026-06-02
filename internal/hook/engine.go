package hook

import (
	"context"
	"fmt"

	"github.com/jesusjhoel/beam/internal/config"
)

type Event string

const (
	EventPreUpload   Event = "pre-upload"
	EventPostUpload  Event = "post-upload"
	EventPreDeploy   Event = "pre-deploy"
	EventPostDeploy  Event = "post-deploy"
	EventOnConnect   Event = "on-connect"
	EventOnError     Event = "on-error"
)

type Engine struct{}

func NewEngine() *Engine {
	return &Engine{}
}

func (e *Engine) Fire(ctx context.Context, event Event, project *config.Project) error {
	_ = ctx
	var actions []config.HookAction
	switch event {
	case EventPreUpload:
		actions = project.Hooks.PreUpload
	case EventPostUpload:
		actions = project.Hooks.PostUpload
	case EventPreDeploy:
		actions = project.Hooks.PreDeploy
	case EventPostDeploy:
		actions = project.Hooks.PostDeploy
	case EventOnConnect:
		actions = project.Hooks.OnConnect
	case EventOnError:
		actions = project.Hooks.OnError
	}
	for _, a := range actions {
		if err := e.run(ctx, a); err != nil && a.Blocking {
			return fmt.Errorf("hook %s action %s: %w", event, a.Type, err)
		}
	}
	return nil
}

func (e *Engine) run(_ context.Context, _ config.HookAction) error {
	return fmt.Errorf("not implemented")
}
