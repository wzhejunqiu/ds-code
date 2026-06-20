package chattool

import (
	"charm.land/lipgloss/v2"
	"github.com/wzhejunqiu/ds-code/internal/ui/theme"
)

var (
	styleTool           = lipgloss.NewStyle().Foreground(theme.Muted)
	styleToolName       = lipgloss.NewStyle().Foreground(theme.Text).Bold(true)
	styleToolCommand    = lipgloss.NewStyle().Foreground(theme.Muted)
	styleToolMeta       = lipgloss.NewStyle().Foreground(theme.Muted)
	styleToolResult     = lipgloss.NewStyle().Foreground(theme.Muted)
	styleToolExpandHint = lipgloss.NewStyle().Foreground(theme.Muted).Italic(true)
	styleToolError      = lipgloss.NewStyle().Foreground(theme.Error)
	styleToolSuccess    = lipgloss.NewStyle().Foreground(theme.Success)
	styleToolShellCmds  = lipgloss.NewStyle().Foreground(theme.Muted)
)
