package hook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/codigoreactivo/beam/internal/config"
)

type Event string

const (
	EventPreUpload  Event = "pre-upload"
	EventPostUpload Event = "post-upload"
	EventPreDeploy  Event = "pre-deploy"
	EventPostDeploy Event = "post-deploy"
	EventOnConnect  Event = "on-connect"
	EventOnError    Event = "on-error"
)

type Result struct {
	Type     string
	Label    string
	ExitCode int
	Status   int
	Duration time.Duration
	Err      error
}

func (r Result) OK() bool { return r.Err == nil }

type Engine struct {
	client *http.Client
}

func NewEngine() *Engine {
	return &Engine{client: &http.Client{Timeout: 30 * time.Second}}
}

// Fire runs all hooks registered for the given event on the project.
// If a blocking hook fails, execution stops and the error is returned.
func (e *Engine) Fire(ctx context.Context, event Event, p *config.Project) ([]Result, error) {
	actions := actionsFor(event, p)
	results := make([]Result, 0, len(actions))
	for _, a := range actions {
		r := e.run(ctx, a)
		results = append(results, r)
		if r.Err != nil && a.Blocking {
			return results, fmt.Errorf("hook %s blocked: %w", event, r.Err)
		}
	}
	return results, nil
}

func (e *Engine) run(ctx context.Context, a config.HookAction) Result {
	start := time.Now()
	r := Result{Type: a.Type}

	switch a.Type {
	case "shell":
		r.Label = a.Cmd
		r.ExitCode, r.Err = e.runShell(ctx, a.Cmd)
	case "webhook":
		r.Label = a.URL
		r.Status, r.Err = e.runWebhook(ctx, a)
	case "notification":
		r.Label = a.Title
		r.Err = e.notify(a)
	default:
		r.Err = fmt.Errorf("unknown hook type %q", a.Type)
	}

	r.Duration = time.Since(start)
	return r
}

func (e *Engine) runShell(ctx context.Context, cmdStr string) (int, error) {
	cmd := exec.CommandContext(ctx, "sh", "-c", cmdStr)
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode(), fmt.Errorf("exit %d: %s", exitErr.ExitCode(), strings.TrimSpace(string(exitErr.Stderr)))
		}
		return -1, err
	}
	return 0, nil
}

func (e *Engine) runWebhook(ctx context.Context, a config.HookAction) (int, error) {
	method := a.Method
	if method == "" {
		method = http.MethodPost
	}
	body := []byte(a.Body)
	if len(body) == 0 {
		body, _ = json.Marshal(map[string]string{"event": "deploy"})
	}

	req, err := http.NewRequestWithContext(ctx, method, a.URL, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return 0, err
	}
	resp.Body.Close()

	if resp.StatusCode >= 400 {
		return resp.StatusCode, fmt.Errorf("webhook returned HTTP %d", resp.StatusCode)
	}
	return resp.StatusCode, nil
}

func (e *Engine) notify(a config.HookAction) error {
	// Desktop notification — platform-specific, best-effort
	title := a.Title
	if title == "" {
		title = "Beam"
	}
	msg := a.Message
	cmd := exec.Command("osascript", "-e",
		fmt.Sprintf(`display notification %q with title %q`, msg, title))
	cmd.Run() // ignore error — notification is non-blocking by design
	return nil
}

func actionsFor(event Event, p *config.Project) []config.HookAction {
	switch event {
	case EventPreUpload:
		return p.Hooks.PreUpload
	case EventPostUpload:
		return p.Hooks.PostUpload
	case EventPreDeploy:
		return p.Hooks.PreDeploy
	case EventPostDeploy:
		return p.Hooks.PostDeploy
	case EventOnConnect:
		return p.Hooks.OnConnect
	case EventOnError:
		return p.Hooks.OnError
	default:
		return nil
	}
}
