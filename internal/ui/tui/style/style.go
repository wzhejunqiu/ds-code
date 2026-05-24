package style

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/wzhejunqiu/ds-code/internal/ui/theme"
)

var (
	App = lipgloss.NewStyle().Foreground(theme.Text)

	HeaderTitle = lipgloss.NewStyle().Foreground(theme.Text).Bold(true)
	HeaderMeta  = lipgloss.NewStyle().Foreground(theme.Muted)
	HeaderPath  = lipgloss.NewStyle().Foreground(theme.Muted)

	Logo = lipgloss.NewStyle().Foreground(theme.DeepSeek).Bold(true)

	Divider = lipgloss.NewStyle().Foreground(theme.Divider)

	InputPrompt = lipgloss.NewStyle().Foreground(theme.Text).Bold(true)
	InputText   = lipgloss.NewStyle().Foreground(theme.Text)

	FooterHint   = lipgloss.NewStyle().Foreground(theme.Muted)
	FooterStat   = lipgloss.NewStyle().Foreground(theme.Muted)
	FooterAccent = lipgloss.NewStyle().Foreground(theme.Accent)

	Overlay = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.OverlayBd).
		Foreground(theme.Text).
		Padding(0, 1)
)

// LogoArt is the ASCII whale logo (scripts/img2logo).
const LogoArt = `        ▄▄▄ ▄▄▄▄▄    ▄█
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
