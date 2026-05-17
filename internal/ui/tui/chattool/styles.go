package chattool

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/hejunqiu/ds-code/internal/ui/theme"
)

var (
	styleTool           = lipgloss.NewStyle().Foreground(theme.Muted)
	styleToolTitle      = lipgloss.NewStyle().Foreground(theme.Accent).Bold(true)
	styleToolName       = lipgloss.NewStyle().Foreground(theme.Text).Bold(true)
	styleToolCommand    = lipgloss.NewStyle().Foreground(theme.Muted)
	styleToolMeta       = lipgloss.NewStyle().Foreground(theme.Muted)
	styleToolResult     = lipgloss.NewStyle().Foreground(theme.Muted)
	styleToolExpandHint = lipgloss.NewStyle().Foreground(theme.Muted).Italic(true)
	styleToolError      = lipgloss.NewStyle().Foreground(theme.Error)
)
