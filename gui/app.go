package gui

import (
	"context"
	"fmt"

	"github.com/codigoreactivo/beam/internal/config"
	"github.com/codigoreactivo/beam/internal/exporter"
	"github.com/codigoreactivo/beam/internal/hook"
	"github.com/codigoreactivo/beam/internal/importer"
	"github.com/codigoreactivo/beam/internal/project"
	beamsync "github.com/codigoreactivo/beam/internal/sync"
	"github.com/codigoreactivo/beam/internal/transfer"
)

// ── result types (JSON-serializable for Wails bindings) ──────────────────────

type Result struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

type ProjectInfo struct {
	config.Project
	// runtime state (populated by App, not stored)
	Connected bool `json:"connected"`
	Watching  bool `json:"watching"`
}

type TestResult struct {
	OK        bool   `json:"ok"`
	LatencyMs int64  `json:"latency_ms"`
	Error     string `json:"error,omitempty"`
}

type SyncResult struct {
	OK      bool   `json:"ok"`
	Added   int    `json:"added"`
	Updated int    `json:"updated"`
	Skipped int    `json:"skipped"`
	Bytes   int64  `json:"bytes"`
	Error   string `json:"error,omitempty"`
}

type DeployResult struct {
	OK               bool     `json:"ok"`
	Added            int      `json:"added"`
	Updated          int      `json:"updated"`
	HooksRun         []string `json:"hooks_run"`
	DurationMs       int64    `json:"duration_ms"`
	Error            string   `json:"error,omitempty"`
}

type AddProjectParams struct {
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	Key      string `json:"key"`
	Local    string `json:"local"`
	Remote   string `json:"remote"`
	Env      string `json:"env"`
}

type SpeedTestResult struct {
	OK           bool    `json:"ok"`
	UploadMbps   float64 `json:"upload_mbps"`
	DownloadMbps float64 `json:"download_mbps"`
	UploadMs     int64   `json:"upload_ms"`
	DownloadMs   int64   `json:"download_ms"`
	SizeBytes    int64   `json:"size_bytes"`
	Error        string  `json:"error,omitempty"`
}

type ImportResult struct {
	OK       bool     `json:"ok"`
	Imported int      `json:"imported"`
	Skipped  int      `json:"skipped"`
	Projects []string `json:"projects"`
	Error    string   `json:"error,omitempty"`
}

type ExportResult struct {
	OK      bool   `json:"ok"`
	Content string `json:"content"`
	Ext     string `json:"ext"`
	Error   string `json:"error,omitempty"`
}

type RemoteEntry struct {
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	IsDir   bool   `json:"is_dir"`
	ModTime string `json:"mod_time"`
}

// ── App ───────────────────────────────────────────────────────────────────────

type App struct {
	ctx context.Context
}

func NewApp() *App { return &App{} }

func (a *App) startup(ctx context.Context) { a.ctx = ctx }

// ── project management ────────────────────────────────────────────────────────

func (a *App) GetProjects() []ProjectInfo {
	mgr, err := loadMgr()
	if err != nil {
		return nil
	}
	projects := mgr.All()
	result := make([]ProjectInfo, len(projects))
	for i, p := range projects {
		result[i] = ProjectInfo{Project: p}
	}
	return result
}

func (a *App) AddProject(params AddProjectParams) Result {
	mgr, err := loadMgr()
	if err != nil {
		return Result{Error: err.Error()}
	}
	p := config.Project{
		Name:     params.Name,
		Protocol: config.Protocol(params.Protocol),
		Host:     params.Host,
		Port:     params.Port,
		User:     params.User,
		Password: params.Password,
		Key:      params.Key,
		Local:    params.Local,
		Remote:   params.Remote,
		Env:      params.Env,
	}
	if p.Protocol == "" {
		p.Protocol = config.ProtocolSFTP
	}
	if p.Port == 0 {
		p.Port = p.DefaultPort()
	}
	if err := mgr.Add(p); err != nil {
		return Result{Error: err.Error()}
	}
	return Result{OK: true}
}

func (a *App) UpdateProject(params AddProjectParams) Result {
	mgr, err := loadMgr()
	if err != nil {
		return Result{Error: err.Error()}
	}
	p := config.Project{
		Name:     params.Name,
		Protocol: config.Protocol(params.Protocol),
		Host:     params.Host,
		Port:     params.Port,
		User:     params.User,
		Password: params.Password,
		Key:      params.Key,
		Local:    params.Local,
		Remote:   params.Remote,
		Env:      params.Env,
	}
	if p.Protocol == "" {
		p.Protocol = config.ProtocolSFTP
	}
	if p.Port == 0 {
		p.Port = p.DefaultPort()
	}
	if err := mgr.Update(p); err != nil {
		return Result{Error: err.Error()}
	}
	return Result{OK: true}
}

func (a *App) RemoveProject(name string) Result {
	mgr, err := loadMgr()
	if err != nil {
		return Result{Error: err.Error()}
	}
	if err := mgr.Remove(name); err != nil {
		return Result{Error: err.Error()}
	}
	return Result{OK: true}
}

func (a *App) TestProject(name string) TestResult {
	mgr, err := loadMgr()
	if err != nil {
		return TestResult{Error: err.Error()}
	}
	p, err := mgr.Get(name)
	if err != nil {
		return TestResult{Error: err.Error()}
	}

	start := now()
	client, err := transfer.Connect(p)
	if err != nil {
		return TestResult{Error: err.Error(), LatencyMs: elapsed(start)}
	}
	defer client.Close()

	if _, err := client.List(a.ctx, p.Remote); err != nil {
		return TestResult{Error: err.Error(), LatencyMs: elapsed(start)}
	}
	return TestResult{OK: true, LatencyMs: elapsed(start)}
}

// ── file operations ───────────────────────────────────────────────────────────

func (a *App) ListRemote(projectName, remotePath string) []RemoteEntry {
	p, client, err := dialProject(projectName)
	if err != nil {
		return nil
	}
	defer client.Close()

	if remotePath == "" {
		remotePath = p.Remote
	}
	entries, err := client.List(a.ctx, remotePath)
	if err != nil {
		return nil
	}
	result := make([]RemoteEntry, len(entries))
	for i, e := range entries {
		result[i] = RemoteEntry{
			Name:    e.Name,
			Size:    e.Size,
			IsDir:   e.IsDir,
			ModTime: e.ModTime.Format("2006-01-02 15:04"),
		}
	}
	return result
}

func (a *App) SyncProject(projectName string, dryRun bool) SyncResult {
	p, client, err := dialProject(projectName)
	if err != nil {
		return SyncResult{Error: err.Error()}
	}
	defer client.Close()

	plan, err := beamsync.Build(a.ctx, client, p)
	if err != nil {
		return SyncResult{Error: err.Error()}
	}
	if !dryRun {
		if err := beamsync.Execute(a.ctx, client, plan, p, nil); err != nil {
			return SyncResult{Error: err.Error()}
		}
	}
	return SyncResult{
		OK:      true,
		Added:   plan.AddCount,
		Updated: plan.UpdateCount,
		Skipped: plan.SkipCount,
		Bytes:   plan.TotalBytes(),
	}
}

func (a *App) DeployProject(projectName string) DeployResult {
	p, client, err := dialProject(projectName)
	if err != nil {
		return DeployResult{Error: err.Error()}
	}
	defer client.Close()

	hooks := hook.NewEngine()
	start := now()
	hooksRun := []string{}

	preResults, err := hooks.Fire(a.ctx, hook.EventPreDeploy, p)
	for _, r := range preResults {
		hooksRun = append(hooksRun, fmt.Sprintf("pre: %s", r.Label))
	}
	if err != nil {
		return DeployResult{Error: fmt.Sprintf("pre-deploy: %s", err)}
	}

	plan, err := beamsync.Build(a.ctx, client, p)
	if err != nil {
		return DeployResult{Error: err.Error()}
	}
	if err := beamsync.Execute(a.ctx, client, plan, p, nil); err != nil {
		return DeployResult{Error: err.Error()}
	}

	postResults, err := hooks.Fire(a.ctx, hook.EventPostDeploy, p)
	for _, r := range postResults {
		hooksRun = append(hooksRun, fmt.Sprintf("post: %s", r.Label))
	}
	if err != nil {
		return DeployResult{Error: fmt.Sprintf("post-deploy: %s", err)}
	}

	return DeployResult{
		OK:         true,
		Added:      plan.AddCount,
		Updated:    plan.UpdateCount,
		HooksRun:   hooksRun,
		DurationMs: elapsed(start),
	}
}

func (a *App) SpeedTest(projectName string) SpeedTestResult {
	_, client, err := dialProject(projectName)
	if err != nil {
		return SpeedTestResult{Error: err.Error()}
	}
	defer client.Close()

	const size = 1 * 1024 * 1024 // 1 MB

	uploadMs, downloadMs, err := client.Benchmark(a.ctx, size)
	if err != nil {
		return SpeedTestResult{Error: err.Error()}
	}

	mbps := func(ms int64) float64 {
		if ms == 0 {
			return 0
		}
		return float64(size) / float64(ms) * 1000 / 1024 / 1024 * 8
	}
	round2 := func(v float64) float64 { return float64(int(v*100+0.5)) / 100 }

	return SpeedTestResult{
		OK:           true,
		UploadMbps:   round2(mbps(uploadMs)),
		DownloadMbps: round2(mbps(downloadMs)),
		UploadMs:     uploadMs,
		DownloadMs:   downloadMs,
		SizeBytes:    size,
	}
}

func (a *App) GetLogs(projectName string, n int) []string {
	if n <= 0 {
		n = 50
	}
	lines, _ := readLastLines(projectName, n)
	return lines
}

func (a *App) GetConfig() *config.GlobalConfig {
	cfg, _ := config.Load()
	return cfg
}

// ── import / export ───────────────────────────────────────────────────────────

// ImportProjects parses a config file (content + original filename for format
// detection) and adds the projects. overwrite=true replaces existing projects.
func (a *App) ImportProjects(filename, content string, overwrite bool) ImportResult {
	projects, err := importer.ParseContent(filename, []byte(content))
	if err != nil {
		return ImportResult{Error: err.Error()}
	}

	mgr, err := loadMgr()
	if err != nil {
		return ImportResult{Error: err.Error()}
	}

	imported, skipped := 0, 0
	var names []string
	for _, p := range projects {
		if p.Port == 0 {
			p.Port = p.DefaultPort()
		}
		addErr := mgr.Add(p)
		if addErr != nil {
			if overwrite {
				_ = mgr.Remove(p.Name)
				_ = mgr.Add(p)
				imported++
				names = append(names, p.Name)
			} else {
				skipped++
			}
			continue
		}
		imported++
		names = append(names, p.Name)
	}
	return ImportResult{OK: true, Imported: imported, Skipped: skipped, Projects: names}
}

// ExportProjects serializes projects to the requested format.
// name="" exports all projects. format: "beam" | "filezilla" | "env".
func (a *App) ExportProjects(name, format string) ExportResult {
	mgr, err := loadMgr()
	if err != nil {
		return ExportResult{Error: err.Error()}
	}

	var list []config.Project
	if name != "" {
		p, err := mgr.Get(name)
		if err != nil {
			return ExportResult{Error: err.Error()}
		}
		list = []config.Project{*p}
	} else {
		list = mgr.All()
	}

	data, err := exporter.Marshal(list, format)
	if err != nil {
		return ExportResult{Error: err.Error()}
	}
	return ExportResult{OK: true, Content: string(data), Ext: exporter.Ext(format)}
}

// ── helpers ───────────────────────────────────────────────────────────────────

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
	client, err := transfer.Connect(p)
	if err != nil {
		return nil, nil, fmt.Errorf("connect to %q: %w", p.Name, err)
	}
	return p, client, nil
}

func readLastLines(proj string, n int) ([]string, error) {
	logPath := config.LogsDir() + "/" + proj + ".log"
	data, err := readFile(logPath)
	if err != nil {
		return []string{}, nil
	}
	lines := splitLines(data)
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines, nil
}
