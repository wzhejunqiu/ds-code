package chat

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/hejunqiu/ds-code/internal/ui/theme"
)

var (
	styleUserBg     = lipgloss.NewStyle().Background(theme.UserBg).Foreground(theme.Text)
	styleBullet     = lipgloss.NewStyle().Foreground(theme.DeepSeek).Bold(true)
	styleBody       = lipgloss.NewStyle().Foreground(theme.Text)
	styleReason     = lipgloss.NewStyle().Foreground(theme.Muted).Italic(true)
	styleTurnMeta   = lipgloss.NewStyle().Foreground(theme.Muted)
	styleInterrupt  = lipgloss.NewStyle().Foreground(theme.Error).Italic(true)
)
