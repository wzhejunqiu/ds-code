package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type chatBlock struct {
	role          string // user | assistant
	content       strings.Builder
	reasoning     strings.Builder
	reasoningOpen bool
	streaming     bool
}

func (b *chatBlock) appendContent(s string) {
	b.content.WriteString(s)
}

func (b *chatBlock) appendReasoning(s string) {
	b.reasoning.WriteString(s)
}

func renderChat(blocks []chatBlock, width int) string {
	if width < 20 {
		width = 20
	}
	var lines []string
	userStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
	asstStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	reasonStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true)

	for _, b := range blocks {
		switch b.role {
		case "user":
			header := userStyle.Render("You")
			body := wrapText(b.content.String(), width-2)
			lines = append(lines, header, body, "")
		case "assistant":
			header := asstStyle.Render("Assistant")
			lines = append(lines, header)
			if b.reasoning.Len() > 0 {
				label := "▸ thinking"
				if b.reasoningOpen {
					label = "▾ thinking"
				}
				lines = append(lines, reasonStyle.Render(label))
				if b.reasoningOpen {
					lines = append(lines, wrapText(b.reasoning.String(), width-2))
				}
			}
			if b.content.Len() > 0 {
				lines = append(lines, wrapText(b.content.String(), width-2))
			} else if b.streaming {
				lines = append(lines, reasonStyle.Render("…"))
			}
			lines = append(lines, "")
		}
	}
	return strings.TrimRight(strings.Join(lines, "\n"), "\n")
}

func wrapText(s string, width int) string {
	if width <= 0 {
		return s
	}
	var out []string
	for _, line := range strings.Split(s, "\n") {
		for len(line) > width {
			out = append(out, line[:width])
			line = line[width:]
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func toolLine(name, preview string, running bool) string {
	if running {
		return fmt.Sprintf("→ %s …", name)
	}
	if preview == "" {
		return fmt.Sprintf("✓ %s", name)
	}
	return fmt.Sprintf("✓ %s  %s", name, truncate(preview, 80))
}
