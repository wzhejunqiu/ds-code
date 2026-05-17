package tui

import "github.com/charmbracelet/lipgloss"

// Claude Code–inspired light theme.
var (
	colorText      = lipgloss.Color("#3D3D3D")
	colorMuted     = lipgloss.Color("#8A8A8A")
	colorAccent      = lipgloss.Color("#D97706")
	colorDeepSeek    = lipgloss.Color("#4D6BFE") // DeepSeek brand blue
	colorDivider   = lipgloss.Color("#C8C4B8")
	colorUserBg    = lipgloss.Color("#E8E6E0")
	colorError     = lipgloss.Color("#C2410C")
)

var (
	styleApp = lipgloss.NewStyle().Foreground(colorText)

	styleHeaderTitle = lipgloss.NewStyle().Foreground(colorText).Bold(true)
	styleHeaderMeta  = lipgloss.NewStyle().Foreground(colorMuted)
	styleHeaderPath  = lipgloss.NewStyle().Foreground(colorMuted)

	styleLogo = lipgloss.NewStyle().Foreground(colorDeepSeek).Bold(true)

	styleDivider = lipgloss.NewStyle().Foreground(colorDivider)

	styleInputPrompt = lipgloss.NewStyle().Foreground(colorText).Bold(true)
	styleInputText   = lipgloss.NewStyle().Foreground(colorText)

	styleFooterHint   = lipgloss.NewStyle().Foreground(colorMuted)
	styleFooterStat   = lipgloss.NewStyle().Foreground(colorMuted)
	styleFooterAccent = lipgloss.NewStyle().Foreground(colorAccent)

	styleChatUserBg = lipgloss.NewStyle().Background(colorUserBg).Foreground(colorText)
	styleChatBullet = lipgloss.NewStyle().Foreground(colorDeepSeek).Bold(true)
	styleChatBody      = lipgloss.NewStyle().Foreground(colorText)
	styleChatReason    = lipgloss.NewStyle().Foreground(colorMuted).Italic(true)
	styleChatTurnMeta    = lipgloss.NewStyle().Foreground(colorMuted)
	styleChatInterrupt   = lipgloss.NewStyle().Foreground(colorError).Italic(true)
	styleChatTool         = lipgloss.NewStyle().Foreground(colorMuted)
	styleChatToolTitle    = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	styleChatToolMeta     = lipgloss.NewStyle().Foreground(colorMuted)
	styleChatToolResult   = lipgloss.NewStyle().Foreground(colorText)
	styleChatToolError    = lipgloss.NewStyle().Foreground(colorError)

	styleOverlay = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#B8B4A8")).
			Foreground(colorText).
			Padding(0, 1)
)

// Pixel whale — DeepSeek 官网吉祥物（面朝左、圆身、右尾、喷水与尾鳍）.
const logoArt = `      ▄
    ▄▀▀▄
  ▄█████▀
 ▄██▀▀▀▀██
 █▀●    ▀█
  ▀▀   ▀▀▀`
