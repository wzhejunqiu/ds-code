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
	styleChatBody   = lipgloss.NewStyle().Foreground(theme.Text)
	styleChatReason         = lipgloss.NewStyle().Foreground(theme.Muted).Italic(true)
	styleChatTurnMeta       = lipgloss.NewStyle().Foreground(theme.Muted)
	styleChatInterrupt = lipgloss.NewStyle().Foreground(theme.Error).Italic(true)

	styleOverlay = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.OverlayBd).
			Foreground(theme.Text).
			Padding(0, 1)
)

// Pixel whale — 由 DeepSeek 官网吉祥物 PNG 半块字符采样生成（scripts/img2logo）.
const logoArt = `        ▄▄▄ ▄▄▄▄▄    ▄█
   ▄▄███████████▄    ████▄ ▄▄▄▄█
 ▄████████████████▄   █████████▀
 ███████████████████▄▄ ▀█████▀
██▀    ▀▀████████▀▀████████
███        ▀███████●▀█████▀
▀██▄         ▀█████▄▄█████
 ███▄          ██████████
  ▀███▄   ▀█▄▄  ▀██████▀
    ▀████▄▄████▄▄▄███████▄
      ▀▀▀████████▀▀`
