package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/hejunqiu/ds-code/internal/ui/theme"
)

var (
	styleApp = lipgloss.NewStyle().Foreground(theme.Text)

	styleHeaderTitle = lipgloss.NewStyle().Foreground(theme.Text).Bold(true)
	styleHeaderMeta  = lipgloss.NewStyle().Foreground(theme.Muted)
	styleHeaderPath  = lipgloss.NewStyle().Foreground(theme.Muted)

	styleLogo = lipgloss.NewStyle().Foreground(theme.DeepSeek).Bold(true)

	styleDivider = lipgloss.NewStyle().Foreground(theme.Divider)

	styleInputPrompt = lipgloss.NewStyle().Foreground(theme.Text).Bold(true)
	styleInputText   = lipgloss.NewStyle().Foreground(theme.Text)

	styleFooterHint   = lipgloss.NewStyle().Foreground(theme.Muted)
	styleFooterStat   = lipgloss.NewStyle().Foreground(theme.Muted)
	styleFooterAccent = lipgloss.NewStyle().Foreground(theme.Accent)

	styleChatUserBg = lipgloss.NewStyle().Background(theme.UserBg).Foreground(theme.Text)
	styleChatBullet = lipgloss.NewStyle().Foreground(theme.DeepSeek).Bold(true)
	styleChatBody      = lipgloss.NewStyle().Foreground(theme.Text)
	styleChatReason    = lipgloss.NewStyle().Foreground(theme.Muted).Italic(true)
	styleChatTurnMeta    = lipgloss.NewStyle().Foreground(theme.Muted)
	styleChatInterrupt   = lipgloss.NewStyle().Foreground(theme.Error).Italic(true)
	styleChatTool          = lipgloss.NewStyle().Foreground(theme.Muted)
	styleChatToolTitle     = lipgloss.NewStyle().Foreground(theme.Accent).Bold(true)
	styleChatToolName      = lipgloss.NewStyle().Foreground(theme.Text).Bold(true)
	styleChatToolCommand   = lipgloss.NewStyle().Foreground(theme.Muted)
	styleChatToolMeta      = lipgloss.NewStyle().Foreground(theme.Muted)
	styleChatToolResult    = lipgloss.NewStyle().Foreground(theme.Muted)
	styleChatToolExpandHint = lipgloss.NewStyle().Foreground(theme.Muted).Italic(true)
	styleChatToolError     = lipgloss.NewStyle().Foreground(theme.Error)

	styleOverlay = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.OverlayBd).
			Foreground(theme.Text).
			Padding(0, 1)
)

// Pixel whale — DeepSeek 官网吉祥物（面朝左、圆身、右尾、喷水与尾鳍）.
const logoArt = `      ▄
    ▄▀▀▄
  ▄█████▀
 ▄██▀▀▀▀██
 █▀●    ▀█
  ▀▀   ▀▀▀`
