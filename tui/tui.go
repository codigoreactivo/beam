package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jesusjhoel/beam/internal/config"
	"github.com/jesusjhoel/beam/internal/project"
	beamsync "github.com/jesusjhoel/beam/internal/sync"
	"github.com/jesusjhoel/beam/internal/transfer"
	"github.com/jesusjhoel/beam/internal/watcher"
)

const leftPanelWidth = 26

// ── messages ──────────────────────────────────────────────────────────────────

type projectsLoadedMsg []config.Project
type errMsg struct{ err error }

type syncResultMsg struct {
	project         string
	added, updated, skipped int
	err             error
}

type logLineMsg struct {
	project string
	ts      time.Time
	line    string
	level   string // success | error | warn | info
}

type watchStartedMsg struct {
	project string
	events  <-chan watcher.ChangeEvent
	cancel  context.CancelFunc
	err     error
}

type watchEventMsg struct {
	project string
	relPath string
	bytes   int64
	err     error
	events  <-chan watcher.ChangeEvent // carry channel forward
}

type watchStoppedMsg struct{ project string }

// ── project state ─────────────────────────────────────────────────────────────

type projectState struct {
	syncing  bool
	watching bool
	cancel   context.CancelFunc
	events   <-chan watcher.ChangeEvent
}

// ── model ─────────────────────────────────────────────────────────────────────

type Model struct {
	projects      []config.Project
	cursor        int
	states        map[string]*projectState
	logLines      map[string][]logLineMsg
	width, height int
	ready         bool
	spinner       spinner.Model
	viewport      viewport.Model
	showHelp      bool
	globalErr     string
}

func initialModel() Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(colorAccent)
	return Model{
		states:   map[string]*projectState{},
		logLines: map[string][]logLineMsg{},
		spinner:  sp,
	}
}

func (m *Model) currentProject() *config.Project {
	if len(m.projects) == 0 || m.cursor >= len(m.projects) {
		return nil
	}
	return &m.projects[m.cursor]
}

func (m *Model) state(name string) *projectState {
	if s, ok := m.states[name]; ok {
		return s
	}
	s := &projectState{}
	m.states[name] = s
	return s
}

func (m *Model) addLog(proj, line, level string) {
	entry := logLineMsg{project: proj, ts: time.Now(), line: line, level: level}
	m.logLines[proj] = append(m.logLines[proj], entry)
	if len(m.logLines[proj]) > 200 {
		m.logLines[proj] = m.logLines[proj][len(m.logLines[proj])-200:]
	}
}

func (m *Model) isAnyBusy() bool {
	for _, s := range m.states {
		if s.syncing {
			return true
		}
	}
	return false
}

func (m *Model) refreshViewport() {
	m.viewport.SetContent(m.renderLog())
	m.viewport.GotoBottom()
}

// ── init ──────────────────────────────────────────────────────────────────────

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, loadProjectsCmd())
}

func loadProjectsCmd() tea.Cmd {
	return func() tea.Msg {
		cfg, err := config.Load()
		if err != nil {
			return errMsg{err}
		}
		mgr, err := project.NewManager(cfg)
		if err != nil {
			return errMsg{err}
		}
		return projectsLoadedMsg(mgr.All())
	}
}

// ── update ────────────────────────────────────────────────────────────────────

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.ready = true
		vpH := m.height - 5
		if vpH < 1 {
			vpH = 1
		}
		vpW := m.width - leftPanelWidth - 4
		if vpW < 10 {
			vpW = 10
		}
		m.viewport = viewport.New(vpW, vpH)
		m.refreshViewport()

	case projectsLoadedMsg:
		m.projects = []config.Project(msg)
		if m.cursor >= len(m.projects) {
			m.cursor = 0
		}
		m.refreshViewport()

	case errMsg:
		m.globalErr = msg.err.Error()

	case syncResultMsg:
		st := m.state(msg.project)
		st.syncing = false
		if msg.err != nil {
			m.addLog(msg.project, "✗ "+msg.err.Error(), "error")
		} else {
			m.addLog(msg.project, fmt.Sprintf(
				"✓ sync done — %d added, %d updated, %d skipped",
				msg.added, msg.updated, msg.skipped), "success")
		}
		m.refreshViewport()

	case watchStartedMsg:
		st := m.state(msg.project)
		if msg.err != nil {
			st.watching = false
			m.addLog(msg.project, "✗ watch failed: "+msg.err.Error(), "error")
		} else {
			st.events = msg.events
			st.cancel = msg.cancel
			m.addLog(msg.project, "◉ watcher started", "info")
			cmds = append(cmds, waitForWatchEventCmd(msg.project, msg.events))
		}
		m.refreshViewport()

	case watchEventMsg:
		if msg.err != nil {
			m.addLog(msg.project, "✗ "+msg.err.Error(), "error")
		} else {
			m.addLog(msg.project,
				fmt.Sprintf("↑ %s (%s)", msg.relPath, formatBytes(msg.bytes)), "success")
		}
		m.refreshViewport()
		// re-subscribe to the next event from the same channel
		if msg.events != nil {
			cmds = append(cmds, waitForWatchEventCmd(msg.project, msg.events))
		}

	case watchStoppedMsg:
		st := m.state(msg.project)
		st.watching = false
		st.events = nil
		st.cancel = nil
		m.addLog(msg.project, "◼ watch stopped", "info")
		m.refreshViewport()

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case tea.KeyMsg:
		if cmd := m.handleKey(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	if m.ready {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) handleKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "q", "ctrl+c":
		return tea.Quit

	case "?":
		m.showHelp = !m.showHelp

	case "tab", "j", "down":
		if len(m.projects) > 0 {
			m.cursor = (m.cursor + 1) % len(m.projects)
			m.refreshViewport()
		}

	case "shift+tab", "k", "up":
		if len(m.projects) > 0 {
			m.cursor = (m.cursor - 1 + len(m.projects)) % len(m.projects)
			m.refreshViewport()
		}

	case "u", "U":
		if p := m.currentProject(); p != nil {
			st := m.state(p.Name)
			if !st.syncing {
				st.syncing = true
				m.addLog(p.Name, "→ syncing...", "info")
				return doSyncCmd(p)
			}
		}

	case "w", "W":
		if p := m.currentProject(); p != nil {
			st := m.state(p.Name)
			if st.watching {
				if st.cancel != nil {
					st.cancel()
				}
				// watchStoppedMsg will arrive async
			} else {
				st.watching = true
				m.addLog(p.Name, fmt.Sprintf("→ starting watcher on %s...", p.Local), "info")
				return startWatchSetupCmd(p)
			}
		}

	case "r", "R":
		return loadProjectsCmd()
	}
	return nil
}

// ── async commands ────────────────────────────────────────────────────────────

func doSyncCmd(p *config.Project) tea.Cmd {
	pCopy := *p
	return func() tea.Msg {
		client, err := transfer.Connect(&pCopy)
		if err != nil {
			return syncResultMsg{project: pCopy.Name, err: err}
		}
		defer client.Close()
		plan, err := beamsync.Build(context.Background(), client, &pCopy)
		if err != nil {
			return syncResultMsg{project: pCopy.Name, err: err}
		}
		if err := beamsync.Execute(context.Background(), client, plan, &pCopy, nil); err != nil {
			return syncResultMsg{project: pCopy.Name, err: err}
		}
		return syncResultMsg{
			project: pCopy.Name,
			added:   plan.AddCount,
			updated: plan.UpdateCount,
			skipped: plan.SkipCount,
		}
	}
}

func startWatchSetupCmd(p *config.Project) tea.Cmd {
	pCopy := *p
	return func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		w, err := watcher.New(&pCopy)
		if err != nil {
			cancel()
			return watchStartedMsg{project: pCopy.Name, err: err}
		}
		if err := w.Start(ctx); err != nil {
			cancel()
			return watchStartedMsg{project: pCopy.Name, err: err}
		}
		return watchStartedMsg{
			project: pCopy.Name,
			events:  w.Events,
			cancel:  cancel,
		}
	}
}

// waitForWatchEventCmd blocks on the events channel and returns one event.
// Update re-calls it after each event so the loop continues indefinitely.
func waitForWatchEventCmd(projName string, events <-chan watcher.ChangeEvent) tea.Cmd {
	return func() tea.Msg {
		e, ok := <-events
		if !ok {
			return watchStoppedMsg{projName}
		}
		// look up project config to get remote path
		cfg, _ := config.Load()
		mgr, _ := project.NewManager(cfg)
		p, err := mgr.Get(projName)
		if err != nil {
			return watchEventMsg{project: projName, relPath: e.RelPath, events: events}
		}
		client, err := transfer.Connect(p)
		if err != nil {
			return watchEventMsg{project: projName, relPath: e.RelPath, err: err, events: events}
		}
		remote := strings.TrimRight(p.Remote, "/") + "/" + e.RelPath
		n, uploadErr := client.Upload(context.Background(), e.Path, remote)
		client.Close()
		return watchEventMsg{
			project: projName,
			relPath: e.RelPath,
			bytes:   n,
			err:     uploadErr,
			events:  events,
		}
	}
}

// ── view ──────────────────────────────────────────────────────────────────────

func (m Model) View() string {
	if !m.ready {
		return "\n  Loading Beam...\n"
	}
	if m.showHelp {
		return m.renderHelp()
	}
	return lipgloss.JoinVertical(lipgloss.Left,
		m.renderHeader(),
		lipgloss.JoinHorizontal(lipgloss.Top, m.renderProjectList(), m.renderRightPanel()),
		m.renderStatus(),
	)
}

func (m Model) renderHeader() string {
	title := styleHeader.Render("  Beam ")
	sub := styleHeaderMuted.Render("SFTP/FTP deploy client")
	pad := styleHeader.Width(m.width - lipgloss.Width(title) - lipgloss.Width(sub)).Render("")
	return lipgloss.JoinHorizontal(lipgloss.Left, title, sub, pad)
}

func (m Model) renderProjectList() string {
	h := m.height - 3
	rows := []string{styleProjectDim.Render(" PROJECTS")}

	for i, p := range m.projects {
		st := m.state(p.Name)
		icon := " ○"
		ns := styleProjectIdle
		if i == m.cursor {
			icon = " ●"
			ns = styleProjectActive
		}
		row := icon + " " + ns.Render(truncate(p.Name, leftPanelWidth-6))
		switch {
		case st.syncing:
			row += " " + styleLogWarn.Render("↻")
		case st.watching:
			row += " " + styleLogSuccess.Render("◉")
		}
		rows = append(rows, row)
	}

	if len(m.projects) == 0 {
		rows = append(rows, "", styleProjectDim.Render(" No projects yet."),
			styleProjectDim.Render(" beam project add <name>"))
	}

	rows = append(rows, "", styleProjectDim.Render(" [R] Reload"))
	for len(rows) < h {
		rows = append(rows, "")
	}
	return styleBorder.Width(leftPanelWidth).Height(h).Render(
		strings.Join(rows[:h], "\n"))
}

func (m Model) renderRightPanel() string {
	h := m.height - 3
	w := m.width - leftPanelWidth - 4

	var header string
	if p := m.currentProject(); p != nil {
		st := m.state(p.Name)
		badge := styleLogMuted.Render("○ idle")
		switch {
		case st.syncing:
			badge = styleLogWarn.Render("↻ syncing")
		case st.watching:
			badge = styleLogSuccess.Render("◉ watching")
		}
		header = fmt.Sprintf(" %s  %s  %s → %s",
			styleProjectActive.Render(p.Name), badge,
			styleLogMuted.Render(p.Local), styleLogMuted.Render(p.Remote))
	} else {
		header = styleProjectDim.Render(" No projects. Add one with: beam project add <name>")
	}

	m.viewport.Width = w - 2
	m.viewport.Height = h - 3

	body := lipgloss.JoinVertical(lipgloss.Left,
		header,
		styleProjectDim.Render(strings.Repeat("─", w-2)),
		styleProjectDim.Render(" ACTIVITY LOG"),
		m.viewport.View(),
	)
	return styleBorder.Width(w).Height(h).Render(body)
}

func (m Model) renderLog() string {
	p := m.currentProject()
	if p == nil {
		return ""
	}
	lines := m.logLines[p.Name]
	if len(lines) == 0 {
		return styleLogMuted.Render(" No activity yet — press U to sync, W to watch")
	}
	var sb strings.Builder
	for _, e := range lines {
		ts := styleLogTime.Render(e.ts.Format("15:04:05") + " ")
		var text string
		switch e.level {
		case "success":
			text = styleLogSuccess.Render(e.line)
		case "error":
			text = styleLogError.Render(e.line)
		case "warn":
			text = styleLogWarn.Render(e.line)
		default:
			text = styleLogMuted.Render(e.line)
		}
		sb.WriteString(" " + ts + text + "\n")
	}
	return sb.String()
}

func (m Model) renderStatus() string {
	keys := [][2]string{
		{"U", "sync"}, {"W", "watch"}, {"Tab", "next"}, {"↑↓", "move"}, {"?", "help"}, {"Q", "quit"},
	}
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = styleStatusKey.Render(k[0]) + " " + k[1]
	}
	left := strings.Join(parts, "  ")

	right := ""
	watching := 0
	for _, s := range m.states {
		if s.watching {
			watching++
		}
	}
	switch {
	case m.isAnyBusy():
		right = m.spinner.View() + " working"
	case watching > 0:
		right = styleLogSuccess.Render(fmt.Sprintf("◉ %d watching", watching))
	case m.globalErr != "":
		right = styleLogError.Render("✗ " + m.globalErr)
	}

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 0 {
		gap = 0
	}
	return styleStatusBar.Width(m.width).Render(
		left + strings.Repeat(" ", gap) + right)
}

func (m Model) renderHelp() string {
	lines := []string{
		styleHeader.Render("  Beam — Keyboard Shortcuts  "),
		"",
		"  " + styleStatusKey.Render("Tab / ↑↓ / j / k") + "  Navigate projects",
		"  " + styleStatusKey.Render("U") + "  Sync selected project",
		"  " + styleStatusKey.Render("W") + "  Toggle file watcher",
		"  " + styleStatusKey.Render("R") + "  Reload project list",
		"  " + styleStatusKey.Render("?") + "  Toggle help",
		"  " + styleStatusKey.Render("Q / Ctrl+C") + "  Quit",
	}
	return strings.Join(lines, "\n")
}

// ── entry point ───────────────────────────────────────────────────────────────

func Run() error {
	_, err := tea.NewProgram(
		initialModel(),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	).Run()
	return err
}

func formatBytes(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1fMB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1fKB", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%dB", n)
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
