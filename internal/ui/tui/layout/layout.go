package layout

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/style"
)

func Divider(width int) string {
	if width < 1 {
		width = 1
	}
	return style.Divider.Render(strings.Repeat("─", width))
}

func InputFrame(width int, inputView string) string {
	prompt := style.InputPrompt.Render("> ")
	line := lipgloss.JoinHorizontal(lipgloss.Top, prompt, style.InputText.Render(inputView))
	var b strings.Builder
	b.WriteString(Divider(width))
	b.WriteString("\n")
	b.WriteString(line)
	b.WriteString("\n")
	b.WriteString(Divider(width))
	return b.String()
}

func Footer(width int, left, right string) string {
	if width < 20 {
		width = 20
	}
	leftStyled := style.FooterHint.Render(left)
	rightStyled := style.FooterStat.Render(right)
	gap := width - lipgloss.Width(leftStyled) - lipgloss.Width(rightStyled)
	if gap < 1 {
		gap = 1
	}
	return leftStyled + strings.Repeat(" ", gap) + rightStyled
}
