// Package theme holds shared lipgloss colors for terminal UI (TUI, pickers, overlays).
// Import these instead of hard-coding hex values so chat, input, and lists stay consistent.
package theme

import "charm.land/lipgloss/v2"

// Claude Code–inspired light palette.
var (
	Text      = lipgloss.Color("#3D3D3D")
	Muted     = lipgloss.Color("#8A8A8A")
	Accent    = lipgloss.Color("#D97706")
	DeepSeek  = lipgloss.Color("#4D6BFE")
	Divider   = lipgloss.Color("#C8C4B8")
	UserBg    = lipgloss.Color("#E8E6E0")
	Error     = lipgloss.Color("#C2410C")
	Success   = lipgloss.Color("#15803D")
	OverlayBd = lipgloss.Color("#B8B4A8")
)
