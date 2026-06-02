package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jesusjhoel/beam/internal/config"
	"github.com/jesusjhoel/beam/internal/hook"
	"github.com/jesusjhoel/beam/internal/project"
	beamsync "github.com/jesusjhoel/beam/internal/sync"
	"github.com/jesusjhoel/beam/internal/transfer"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func Run(_ int) error {
	s := server.NewMCPServer("beam", "0.1.0",
		server.WithToolCapabilities(false),
		server.WithResourceCapabilities(true, false),
	)

	registerTools(s)
	registerResources(s)

	return server.ServeStdio(s)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func ok(v any) *mcpgo.CallToolResult {
	b, _ := json.Marshal(v)
	return mcpgo.NewToolResultText(string(b))
}

func fail(err error) *mcpgo.CallToolResult {
	b, _ := json.Marshal(map[string]string{"error": err.Error()})
	return mcpgo.NewToolResultText(string(b))
}

func args(req mcpgo.CallToolRequest) map[string]any {
	if m, ok := req.Params.Arguments.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

func str(req mcpgo.CallToolRequest, key string) string {
	if v, ok := args(req)[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func boolArg(req mcpgo.CallToolRequest, key string) bool {
	if v, ok := args(req)[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func loadMgr() (*project.Manager, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	return project.NewManager(cfg)
}

func dialProject(name string) (*config.Project, transfer.Client, error) {
	mgr, err := loadMgr()
	if err != nil {
		return nil, nil, err
	}
	p, err := mgr.Get(name)
	if err != nil {
		return nil, nil, err
	}
	c, err := transfer.Connect(p)
	if err != nil {
		return nil, nil, fmt.Errorf("connect to %q: %w", p.Name, err)
	}
	return p, c, nil
}

// ── tools ─────────────────────────────────────────────────────────────────────

func registerTools(s *server.MCPServer) {
	// ── project management ────────────────────────────────────────────

	s.AddTool(mcpgo.NewTool("beam_project_list",
		mcpgo.WithDescription("List all configured Beam projects"),
	), func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		mgr, err := loadMgr()
		if err != nil {
			return fail(err), nil
		}
		return ok(map[string]any{"projects": mgr.All()}), nil
	})

	s.AddTool(mcpgo.NewTool("beam_project_add",
		mcpgo.WithDescription("Add a new project profile"),
		mcpgo.WithString("name", mcpgo.Required(), mcpgo.Description("Project name")),
		mcpgo.WithString("protocol", mcpgo.Description("sftp | ftp | ftps")),
		mcpgo.WithString("host", mcpgo.Required(), mcpgo.Description("Remote host")),
		mcpgo.WithString("user", mcpgo.Required(), mcpgo.Description("Remote user")),
		mcpgo.WithString("password", mcpgo.Description("Password (if not using key)")),
		mcpgo.WithString("key_path", mcpgo.Description("Path to SSH private key")),
		mcpgo.WithString("local", mcpgo.Required(), mcpgo.Description("Local directory")),
		mcpgo.WithString("remote", mcpgo.Required(), mcpgo.Description("Remote directory")),
		mcpgo.WithString("env", mcpgo.Description("dev | staging | production")),
	), func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		mgr, err := loadMgr()
		if err != nil {
			return fail(err), nil
		}
		p := config.Project{
			Name:     str(req, "name"),
			Protocol: config.Protocol(str(req, "protocol")),
			Host:     str(req, "host"),
			User:     str(req, "user"),
			Password: str(req, "password"),
			Key:      str(req, "key_path"),
			Local:    str(req, "local"),
			Remote:   str(req, "remote"),
			Env:      str(req, "env"),
		}
		if p.Protocol == "" {
			p.Protocol = config.ProtocolSFTP
		}
		if p.Port == 0 {
			p.Port = p.DefaultPort()
		}
		if err := mgr.Add(p); err != nil {
			return fail(err), nil
		}
		return ok(map[string]any{"success": true, "project": p}), nil
	})

	s.AddTool(mcpgo.NewTool("beam_project_remove",
		mcpgo.WithDescription("Remove a project profile"),
		mcpgo.WithString("name", mcpgo.Required(), mcpgo.Description("Project name")),
	), func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		mgr, err := loadMgr()
		if err != nil {
			return fail(err), nil
		}
		if err := mgr.Remove(str(req, "name")); err != nil {
			return fail(err), nil
		}
		return ok(map[string]any{"success": true}), nil
	})

	s.AddTool(mcpgo.NewTool("beam_project_test",
		mcpgo.WithDescription("Test connectivity for a project"),
		mcpgo.WithString("name", mcpgo.Required(), mcpgo.Description("Project name")),
	), func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		name := str(req, "name")
		start := time.Now()
		p, client, err := dialProject(name)
		if err != nil {
			return ok(map[string]any{"reachable": false, "error": err.Error()}), nil
		}
		defer client.Close()
		_, listErr := client.List(ctx, p.Remote)
		latency := time.Since(start).Milliseconds()
		if listErr != nil {
			return ok(map[string]any{"reachable": false, "latency_ms": latency, "error": listErr.Error()}), nil
		}
		return ok(map[string]any{"reachable": true, "latency_ms": latency}), nil
	})

	// ── file operations ───────────────────────────────────────────────

	s.AddTool(mcpgo.NewTool("beam_list",
		mcpgo.WithDescription("List contents of a remote directory"),
		mcpgo.WithString("project", mcpgo.Required(), mcpgo.Description("Project name")),
		mcpgo.WithString("path", mcpgo.Description("Remote path (default: project remote root)")),
	), func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		name := str(req, "project")
		p, client, err := dialProject(name)
		if err != nil {
			return fail(err), nil
		}
		defer client.Close()
		remotePath := str(req, "path")
		if remotePath == "" {
			remotePath = p.Remote
		}
		entries, err := client.List(ctx, remotePath)
		if err != nil {
			return fail(err), nil
		}
		return ok(map[string]any{"entries": entries}), nil
	})

	s.AddTool(mcpgo.NewTool("beam_upload",
		mcpgo.WithDescription("Upload a local file or directory to remote"),
		mcpgo.WithString("project", mcpgo.Required(), mcpgo.Description("Project name")),
		mcpgo.WithString("local", mcpgo.Required(), mcpgo.Description("Local path")),
		mcpgo.WithString("remote", mcpgo.Required(), mcpgo.Description("Remote path")),
	), func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		_, client, err := dialProject(str(req, "project"))
		if err != nil {
			return fail(err), nil
		}
		defer client.Close()
		start := time.Now()
		n, err := client.Upload(ctx, str(req, "local"), str(req, "remote"))
		if err != nil {
			return fail(err), nil
		}
		return ok(map[string]any{
			"success": true, "bytes_transferred": n,
			"duration_ms": time.Since(start).Milliseconds(),
		}), nil
	})

	s.AddTool(mcpgo.NewTool("beam_download",
		mcpgo.WithDescription("Download a remote file to local path"),
		mcpgo.WithString("project", mcpgo.Required(), mcpgo.Description("Project name")),
		mcpgo.WithString("remote", mcpgo.Required(), mcpgo.Description("Remote path")),
		mcpgo.WithString("local", mcpgo.Required(), mcpgo.Description("Local path")),
	), func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		_, client, err := dialProject(str(req, "project"))
		if err != nil {
			return fail(err), nil
		}
		defer client.Close()
		n, err := client.Download(ctx, str(req, "remote"), str(req, "local"))
		if err != nil {
			return fail(err), nil
		}
		return ok(map[string]any{"success": true, "bytes_transferred": n}), nil
	})

	s.AddTool(mcpgo.NewTool("beam_delete",
		mcpgo.WithDescription("Delete a remote file or directory"),
		mcpgo.WithString("project", mcpgo.Required(), mcpgo.Description("Project name")),
		mcpgo.WithString("path", mcpgo.Required(), mcpgo.Description("Remote path")),
	), func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		_, client, err := dialProject(str(req, "project"))
		if err != nil {
			return fail(err), nil
		}
		defer client.Close()
		if err := client.Remove(ctx, str(req, "path")); err != nil {
			return fail(err), nil
		}
		return ok(map[string]any{"success": true}), nil
	})

	s.AddTool(mcpgo.NewTool("beam_move",
		mcpgo.WithDescription("Rename or move a remote file"),
		mcpgo.WithString("project", mcpgo.Required(), mcpgo.Description("Project name")),
		mcpgo.WithString("from", mcpgo.Required(), mcpgo.Description("Source remote path")),
		mcpgo.WithString("to", mcpgo.Required(), mcpgo.Description("Destination remote path")),
	), func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		_, client, err := dialProject(str(req, "project"))
		if err != nil {
			return fail(err), nil
		}
		defer client.Close()
		if err := client.Rename(ctx, str(req, "from"), str(req, "to")); err != nil {
			return fail(err), nil
		}
		return ok(map[string]any{"success": true}), nil
	})

	// ── sync and deploy ───────────────────────────────────────────────

	s.AddTool(mcpgo.NewTool("beam_diff",
		mcpgo.WithDescription("Show diff between local and remote directories"),
		mcpgo.WithString("project", mcpgo.Required(), mcpgo.Description("Project name")),
	), func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		p, client, err := dialProject(str(req, "project"))
		if err != nil {
			return fail(err), nil
		}
		defer client.Close()
		plan, err := beamsync.Build(ctx, client, p)
		if err != nil {
			return fail(err), nil
		}
		toUpload, toUpdate := []string{}, []string{}
		for _, e := range plan.Entries {
			switch e.Action {
			case beamsync.ActionAdd:
				toUpload = append(toUpload, e.RelPath)
			case beamsync.ActionUpdate:
				toUpdate = append(toUpdate, e.RelPath)
			}
		}
		return ok(map[string]any{
			"to_upload": toUpload,
			"to_update": toUpdate,
			"in_sync":   plan.SkipCount,
		}), nil
	})

	s.AddTool(mcpgo.NewTool("beam_sync",
		mcpgo.WithDescription("Sync local directory to remote (incremental)"),
		mcpgo.WithString("project", mcpgo.Required(), mcpgo.Description("Project name")),
		mcpgo.WithBoolean("dry_run", mcpgo.Description("Preview only, no upload")),
	), func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		p, client, err := dialProject(str(req, "project"))
		if err != nil {
			return fail(err), nil
		}
		defer client.Close()
		plan, err := beamsync.Build(ctx, client, p)
		if err != nil {
			return fail(err), nil
		}
		if !boolArg(req, "dry_run") {
			if err := beamsync.Execute(ctx, client, plan, p, nil); err != nil {
				return fail(err), nil
			}
		}
		return ok(map[string]any{
			"uploaded": plan.AddCount,
			"updated":  plan.UpdateCount,
			"skipped":  plan.SkipCount,
			"dry_run":  boolArg(req, "dry_run"),
		}), nil
	})

	s.AddTool(mcpgo.NewTool("beam_deploy",
		mcpgo.WithDescription("Full deploy: pre-hooks → sync → post-hooks"),
		mcpgo.WithString("project", mcpgo.Required(), mcpgo.Description("Project name")),
		mcpgo.WithBoolean("dry_run", mcpgo.Description("Preview only")),
	), func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		p, client, err := dialProject(str(req, "project"))
		if err != nil {
			return fail(err), nil
		}
		defer client.Close()
		start := time.Now()
		hooks := hook.NewEngine()
		hooksRun := []string{}

		preResults, err := hooks.Fire(ctx, hook.EventPreDeploy, p)
		for _, r := range preResults {
			hooksRun = append(hooksRun, fmt.Sprintf("pre-deploy: %s", r.Label))
		}
		if err != nil {
			return fail(fmt.Errorf("pre-deploy: %w", err)), nil
		}

		plan, err := beamsync.Build(ctx, client, p)
		if err != nil {
			return fail(err), nil
		}
		if !boolArg(req, "dry_run") {
			if err := beamsync.Execute(ctx, client, plan, p, nil); err != nil {
				return fail(err), nil
			}
		}

		postResults, err := hooks.Fire(ctx, hook.EventPostDeploy, p)
		for _, r := range postResults {
			hooksRun = append(hooksRun, fmt.Sprintf("post-deploy: %s", r.Label))
		}
		if err != nil {
			return fail(fmt.Errorf("post-deploy: %w", err)), nil
		}

		return ok(map[string]any{
			"success":            true,
			"hooks_run":          hooksRun,
			"files_transferred":  plan.AddCount + plan.UpdateCount,
			"duration_ms":        time.Since(start).Milliseconds(),
		}), nil
	})

	// ── hooks ─────────────────────────────────────────────────────────

	s.AddTool(mcpgo.NewTool("beam_hook_run",
		mcpgo.WithDescription("Manually trigger a hook event for a project"),
		mcpgo.WithString("project", mcpgo.Required(), mcpgo.Description("Project name")),
		mcpgo.WithString("event", mcpgo.Required(), mcpgo.Description("pre-deploy | post-deploy | pre-upload | post-upload | on-error")),
	), func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		mgr, err := loadMgr()
		if err != nil {
			return fail(err), nil
		}
		p, err := mgr.Get(str(req, "project"))
		if err != nil {
			return fail(err), nil
		}
		hooks := hook.NewEngine()
		event := hook.Event(str(req, "event"))
		results, err := hooks.Fire(ctx, event, p)
		if err != nil {
			return fail(err), nil
		}
		type resultItem struct {
			Type     string `json:"type"`
			Label    string `json:"label"`
			ExitCode int    `json:"exit_code,omitempty"`
			Status   int    `json:"status,omitempty"`
			DurationMs int64 `json:"duration_ms"`
			OK       bool   `json:"ok"`
		}
		items := make([]resultItem, len(results))
		for i, r := range results {
			items[i] = resultItem{
				Type: r.Type, Label: r.Label,
				ExitCode: r.ExitCode, Status: r.Status,
				DurationMs: r.Duration.Milliseconds(), OK: r.OK(),
			}
		}
		return ok(map[string]any{"hooks_run": len(items), "results": items}), nil
	})

	// ── logs ──────────────────────────────────────────────────────────

	s.AddTool(mcpgo.NewTool("beam_logs",
		mcpgo.WithDescription("Return recent log entries for a project"),
		mcpgo.WithString("project", mcpgo.Required(), mcpgo.Description("Project name")),
		mcpgo.WithNumber("limit", mcpgo.Description("Max lines to return (default 20)")),
	), func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		name := str(req, "project")
		lines, err := readLastLines(name, 20)
		if err != nil {
			return fail(err), nil
		}
		return ok(map[string]any{"project": name, "entries": lines}), nil
	})
}

// ── resources ─────────────────────────────────────────────────────────────────

func registerResources(s *server.MCPServer) {
	s.AddResource(mcpgo.NewResource(
		"beam://projects",
		"All Beam project profiles",
		mcpgo.WithMIMEType("application/json"),
	), func(ctx context.Context, req mcpgo.ReadResourceRequest) ([]mcpgo.ResourceContents, error) {
		mgr, err := loadMgr()
		if err != nil {
			return nil, err
		}
		b, _ := json.MarshalIndent(mgr.All(), "", "  ")
		return []mcpgo.ResourceContents{
			mcpgo.TextResourceContents{URI: "beam://projects", MIMEType: "application/json", Text: string(b)},
		}, nil
	})

	s.AddResource(mcpgo.NewResource(
		"beam://config",
		"Global Beam configuration",
		mcpgo.WithMIMEType("application/json"),
	), func(ctx context.Context, req mcpgo.ReadResourceRequest) ([]mcpgo.ResourceContents, error) {
		cfg, err := config.Load()
		if err != nil {
			return nil, err
		}
		b, _ := json.MarshalIndent(cfg, "", "  ")
		return []mcpgo.ResourceContents{
			mcpgo.TextResourceContents{URI: "beam://config", MIMEType: "application/json", Text: string(b)},
		}, nil
	})
}

// readLastLines returns the last n lines of the project log file.
func readLastLines(proj string, n int) ([]string, error) {
	logPath := config.LogsDir() + "/" + proj + ".log"
	data, err := os.ReadFile(logPath)
	if os.IsNotExist(err) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines, nil
}
