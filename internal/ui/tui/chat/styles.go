package chat

import (
	"charm.land/lipgloss/v2"
	"github.com/wzhejunqiu/ds-code/internal/ui/theme"
)

var (
	styleUserBg    = lipgloss.NewStyle().Background(theme.UserBg).Foreground(theme.Text)
	styleBullet    = lipgloss.NewStyle().Foreground(theme.DeepSeek).Bold(true)
	styleReason    = lipgloss.NewStyle().Foreground(theme.Muted).Italic(true)
	styleTurnMeta  = lipgloss.NewStyle().Foreground(theme.Muted)
	styleInterrupt = lipgloss.NewStyle().Foreground(theme.Error).Italic(true)
)
