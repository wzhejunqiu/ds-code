package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// chatBlock is one row in the in-memory transcript (rendered by renderChat).
// Streaming turns mutate the tail assistant block; tools are separate rows between sub-rounds.
type chatBlock struct {
	role               chatBlockRole
	content            strings.Builder
	reasoning          strings.Builder
	reasoningOpen      bool
	reasoningStartedAt time.Time
	reasoningEndedAt   time.Time // zero while thinking is in progress
	planningStartedAt  time.Time
	reasoningDuration  time.Duration
	turnDuration       time.Duration
	streaming          bool
	toolName           string
	toolArgs           string
	toolCommand        string
	toolResult         string
	toolRunning        bool
	toolError          bool
	toolExpanded       bool // show args/result (default collapsed in UI)
}

func (b *chatBlock) appendContent(s string) {
	b.content.WriteString(s)
}

func (b *chatBlock) appendReasoning(s string) {
	b.reasoning.WriteString(s)
}

// finalizeReasoning closes the thinking phase for this assistant segment.
func (b *chatBlock) finalizeReasoning(at time.Time) {
	if b.role != chatRoleAssistant || b.reasoningStartedAt.IsZero() || !b.reasoningEndedAt.IsZero() {
		return
	}
	b.reasoningEndedAt = at
	if b.reasoningDuration == 0 {
		d := at.Sub(b.reasoningStartedAt)
		if d > 0 {
			b.reasoningDuration = d
		}
	}
}

const (
	userPrompt             = "> "
	assistantBullet        = "● "
	toolBullet               = "⚙ "
	toolResultConnector      = "└ "
	toolResultMax            = 2000
	toolResultPreviewMax     = 256
	toolResultPreviewLines   = 3
	toolTitleArgsMax         = 80
	planningBullet           = "◦ "
	planningLabel            = "Planning next moves"
	interruptBullet          = "⏹ "
	interruptLabel           = "Turn cancelled (Esc)"
)

// interruptSessionMarker is stored as a system row so /resume restores the marker.
func interruptSessionMarker() string {
	return "[ds-code] " + interruptLabel
}

func renderChat(blocks []chatBlock, width int, now time.Time, showToolDetails bool) string {
	if width < 20 {
		width = 20
	}
	var lines []string

	for _, b := range blocks {
		switch b.role {
		case chatRoleUser:
			lines = append(lines, renderUserBlock(b.content.String(), width)...)
			lines = append(lines, "")
		case chatRoleAssistant:
			indent := lipgloss.Width(assistantBullet)
			if b.reasoning.Len() > 0 || b.reasoningDuration > 0 || !b.reasoningStartedAt.IsZero() {
				label := reasoningBlockLabel(b.reasoningOpen, b.reasoningStartedAt, b.reasoningEndedAt, now, b.reasoningDuration)
				lines = append(lines, styleChatReason.Render(strings.Repeat(" ", indent)+label))
				if b.reasoningOpen && b.reasoning.Len() > 0 {
					lines = append(lines, styleChatReason.Render(wrapText(b.reasoning.String(), width-indent)))
				}
			}
			if b.content.Len() > 0 {
				lines = append(lines, renderAssistantBlock(b.content.String(), width)...)
			} else if b.streaming {
				lines = append(lines, renderAssistantLine(styleChatReason.Render("…")))
			}
			if b.turnDuration > 0 {
				lines = append(lines, styleChatTurnMeta.Render(strings.Repeat(" ", indent)+turnDurationLine(b.turnDuration)))
			}
			lines = append(lines, "")
		case chatRoleTool:
			expanded := showToolDetails || b.toolExpanded
			lines = append(lines, renderToolBlock(b, width, expanded)...)
			lines = append(lines, "")
		case chatRolePlanning:
			indent := lipgloss.Width(planningBullet)
			lines = append(lines, styleChatReason.Render(strings.Repeat(" ", indent)+planningBlockLabel(b.planningStartedAt, now)))
			lines = append(lines, "")
		case chatRoleInterrupt:
			indent := lipgloss.Width(interruptBullet)
			lines = append(lines, styleChatInterrupt.Render(strings.Repeat(" ", indent)+interruptBullet+interruptLabel))
			lines = append(lines, "")
		}
	}
	return strings.TrimRight(strings.Join(lines, "\n"), "\n")
}

func planningBlockLabel(started, now time.Time) string {
	if started.IsZero() {
		return planningBullet + planningLabel
	}
	d := now.Sub(started)
	if d < 0 {
		d = 0
	}
	if d > 0 {
		return planningBullet + planningLabel + "  " + formatThinkingDuration(d)
	}
	return planningBullet + planningLabel
}

func renderToolBlock(b chatBlock, width int, expanded bool) []string {
	indent := lipgloss.Width(toolBullet)
	var body []string

	if b.toolRunning {
		title := b.toolName + " …"
		body = append(body, styleChatToolTitle.Render(toolBullet+title))
		switch {
		case b.toolCommand != "":
			body = append(body, styleChatToolMeta.Render(strings.Repeat(" ", indent)+b.toolCommand))
		case b.toolArgs != "":
			body = append(body, styleChatToolMeta.Render(strings.Repeat(" ", indent)+truncate(b.toolArgs, toolTitleArgsMax)))
		default:
			body = append(body, styleChatToolMeta.Render(strings.Repeat(" ", indent)+"running…"))
		}
		return body
	}

	body = append(body, renderToolTitleLine(b.toolName, b.toolCommand, b.toolArgs, b.toolError))

	if expanded {
		if b.toolArgs != "" {
			body = append(body, styleChatToolMeta.Render(strings.Repeat(" ", indent)+"args: "+b.toolArgs))
		}
		if b.toolCommand != "" {
			body = append(body, styleChatToolMeta.Render(strings.Repeat(" ", indent)+"command: "+b.toolCommand))
		}
		if b.toolResult != "" {
			body = append(body, renderToolResultLines(truncate(b.toolResult, toolResultMax), indent, width, b.toolError)...)
		}
		return body
	}

	body = append(body, renderToolResultCollapsed(b.toolResult, indent, b.toolError)...)
	return body
}

func renderToolTitleLine(name, command, args string, isError bool) string {
	parts := []string{styleChatToolName.Render(toolBullet + name)}
	if paren := toolParenContent(command, args); paren != "" {
		parts = append(parts, styleChatToolCommand.Render(" ("+paren+")"))
	}
	if isError {
		parts = append(parts, styleChatToolError.Render(" (error)"))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

func toolParenContent(command, args string) string {
	paren := command
	if paren == "" {
		paren = args
		if len(paren) > toolTitleArgsMax {
			paren = truncate(paren, toolTitleArgsMax)
		}
	}
	return paren
}

type toolResultPreviewData struct {
	lines     []string
	moreLines int
	truncated bool
}

func buildToolResultPreview(result string) toolResultPreviewData {
	if result == "" {
		return toolResultPreviewData{}
	}
	totalLines := countResultLines(result)
	truncated := len(result) > toolResultPreviewMax
	s := result
	if truncated {
		s = truncateTo(result, toolResultPreviewMax)
	}
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	shown := len(lines)
	if shown > toolResultPreviewLines {
		lines = lines[:toolResultPreviewLines]
		shown = toolResultPreviewLines
	}
	moreLines := totalLines - shown
	if moreLines < 0 {
		moreLines = 0
	}
	return toolResultPreviewData{lines: lines, moreLines: moreLines, truncated: truncated}
}

func countResultLines(s string) int {
	if s == "" {
		return 0
	}
	return len(strings.Split(strings.TrimRight(s, "\n"), "\n"))
}

func toolExpandHint(moreLines int, truncated bool) string {
	switch {
	case moreLines > 0:
		return fmt.Sprintf("... +%d lines (ctrl+o to expand)", moreLines)
	case truncated:
		return "... (ctrl+o to expand)"
	default:
		return ""
	}
}

func renderToolResultCollapsed(result string, indent int, isError bool) []string {
	preview := buildToolResultPreview(result)
	if len(preview.lines) == 0 {
		return nil
	}
	lines := renderToolResultLines(strings.Join(preview.lines, "\n"), indent, 0, isError)
	if hint := toolExpandHint(preview.moreLines, preview.truncated); hint != "" {
		connWidth := lipgloss.Width(toolResultConnector)
		lines = append(lines, styleChatToolExpandHint.Render(strings.Repeat(" ", indent+connWidth)+hint))
	}
	return lines
}

func renderToolResultLines(result string, indent, width int, isError bool) []string {
	if result == "" {
		return nil
	}
	style := styleChatToolResult
	if isError {
		style = styleChatToolError
	}
	text := strings.TrimRight(result, "\n")
	if width > 0 {
		connWidth := lipgloss.Width(toolResultConnector)
		text = wrapText(text, width-indent-connWidth)
	}
	raw := strings.Split(text, "\n")
	connWidth := lipgloss.Width(toolResultConnector)
	out := make([]string, 0, len(raw))
	for i, line := range raw {
		pad := strings.Repeat(" ", indent)
		if i == 0 {
			out = append(out, style.Render(pad+toolResultConnector+line))
		} else {
			out = append(out, style.Render(pad+strings.Repeat(" ", connWidth)+line))
		}
	}
	return out
}

func truncateTo(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}

func reasoningBlockLabel(open bool, started, ended, now time.Time, fixed time.Duration) string {
	arrow := "▸"
	if open {
		arrow = "▾"
	}
	var d time.Duration
	if fixed > 0 {
		d = fixed
	} else if !started.IsZero() {
		endAt := ended
		if endAt.IsZero() {
			endAt = now
		}
		d = endAt.Sub(started)
		if d < 0 {
			d = 0
		}
	}
	thinkingDone := fixed > 0 || !ended.IsZero()
	if thinkingDone {
		if d > 0 {
			return arrow + " thought for " + formatThinkingDuration(d)
		}
		return arrow + " thought"
	}
	if d > 0 {
		return arrow + " thinking " + formatThinkingDuration(d)
	}
	return arrow + " thinking"
}

func turnDurationLine(d time.Duration) string {
	return "task took " + formatThinkingDuration(d)
}

const thinkingFineDuration = 10 * time.Second

func formatThinkingDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	if d < thinkingFineDuration {
		tenths := int(d.Round(100*time.Millisecond) / (100 * time.Millisecond))
		whole, frac := tenths/10, tenths%10
		if frac == 0 {
			return fmt.Sprintf("%ds", whole)
		}
		return fmt.Sprintf("%d.%ds", whole, frac)
	}
	s := int(d.Round(time.Second).Seconds())
	if s < 60 {
		return fmt.Sprintf("%ds", s)
	}
	m := s / 60
	s %= 60
	if s == 0 {
		return fmt.Sprintf("%dm", m)
	}
	return fmt.Sprintf("%dm %ds", m, s)
}

func renderUserBlock(content string, width int) []string {
	return renderHighlightedBlock(userPrompt, content, width, lipgloss.Width(userPrompt))
}

func renderAssistantBlock(content string, width int) []string {
	return renderMarkdownPrefixedBlock(assistantBullet, styleChatBullet, content, width, lipgloss.Width(assistantBullet))
}

func renderHighlightedBlock(prefix, content string, width, indent int) []string {
	wrapped := wrapText(strings.TrimRight(content, "\n"), width-indent)
	if wrapped == "" {
		return nil
	}
	lines := strings.Split(wrapped, "\n")
	out := make([]string, 0, len(lines))
	for i, line := range lines {
		p := prefix
		if i > 0 {
			p = strings.Repeat(" ", indent)
		}
		out = append(out, renderHighlightedLine(p, line, width))
	}
	return out
}

func renderPlainPrefixedBlock(prefix string, prefixStyle lipgloss.Style, content string, width, indent int) []string {
	wrapped := wrapText(strings.TrimRight(content, "\n"), width-indent)
	if wrapped == "" {
		return nil
	}
	lines := strings.Split(wrapped, "\n")
	out := make([]string, 0, len(lines))
	for i, line := range lines {
		p, ps := prefix, prefixStyle
		if i > 0 {
			p = strings.Repeat(" ", indent)
			ps = lipgloss.NewStyle()
		}
		out = append(out, renderPlainPrefixedLine(p, ps, styleChatBody.Render(line)))
	}
	return out
}

func renderAssistantLine(body string) string {
	return renderPlainPrefixedLine(assistantBullet, styleChatBullet, body)
}

func renderHighlightedLine(prefix, text string, width int) string {
	return styleChatUserBg.Width(width).Align(lipgloss.Left).Render(prefix + text)
}

func renderPlainPrefixedLine(prefix string, prefixStyle lipgloss.Style, body string) string {
	if prefix == "" {
		return body
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, prefixStyle.Render(prefix), body)
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

func toolLine(name, args, command, preview string, running, isError bool) string {
	var s string
	switch {
	case running:
		s = fmt.Sprintf("→ %s …", name)
	case isError:
		s = fmt.Sprintf("✗ %s", name)
	default:
		s = fmt.Sprintf("✓ %s", name)
	}
	if command != "" {
		s += "  " + truncate(command, 60)
	} else if args != "" {
		s += "  " + truncate(args, 60)
	}
	if !running && preview != "" && preview != "…" {
		s += "  " + truncate(preview, 80)
	}
	if isError {
		return styleChatToolError.Render(s)
	}
	return styleChatTool.Render(s)
}
