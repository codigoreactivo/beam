package tui

import "github.com/charmbracelet/lipgloss"

var (
	colorAccent  = lipgloss.Color("#7C3AED") // violet
	colorMuted   = lipgloss.Color("#6B7280")
	colorSuccess = lipgloss.Color("#10B981")
	colorError   = lipgloss.Color("#EF4444")
	colorWarn    = lipgloss.Color("#F59E0B")
	colorBorder  = lipgloss.Color("#374151")
	colorBg      = lipgloss.Color("#111827")
	colorText    = lipgloss.Color("#F9FAFB")
	colorSubtext = lipgloss.Color("#9CA3AF")

	styleHeader = lipgloss.NewStyle().
			Background(colorAccent).
			Foreground(colorText).
			Bold(true).
			Padding(0, 1)

	styleHeaderMuted = lipgloss.NewStyle().
				Background(colorAccent).
				Foreground(lipgloss.Color("#C4B5FD")).
				Padding(0, 1)

	styleBorder = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(colorBorder)

	styleProjectActive = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorAccent)

	styleProjectIdle = lipgloss.NewStyle().
				Foreground(colorText)

	styleProjectDim = lipgloss.NewStyle().
			Foreground(colorMuted)

	styleStatusBar = lipgloss.NewStyle().
			Background(lipgloss.Color("#1F2937")).
			Foreground(colorSubtext).
			Padding(0, 1)

	styleStatusKey = lipgloss.NewStyle().
			Background(lipgloss.Color("#374151")).
			Foreground(colorText).
			Padding(0, 1)

	styleLogSuccess = lipgloss.NewStyle().Foreground(colorSuccess)
	styleLogError   = lipgloss.NewStyle().Foreground(colorError)
	styleLogWarn    = lipgloss.NewStyle().Foreground(colorWarn)
	styleLogMuted   = lipgloss.NewStyle().Foreground(colorMuted)
	styleLogTime    = lipgloss.NewStyle().Foreground(colorSubtext)
)
