package chat

import (
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/chattool"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/markdown"
)

// Render formats chat blocks for the main transcript viewport.
func Render(blocks []Block, width int, now time.Time, showToolDetails bool) string {
	if width < 20 {
		width = 20
	}
	var lines []string

	for i := range blocks {
		b := &blocks[i]
		switch b.Role {
		case RoleUser:
			lines = append(lines, renderUserBlock(b.Content, width)...)
			lines = append(lines, "")
		case RoleAssistant:
			indent := lipgloss.Width(assistantBullet)
			if b.Reasoning != "" || b.ReasoningDuration > 0 || !b.ReasoningStartedAt.IsZero() {
				expanded := reasoningExpanded(b)
				label := reasoningBlockLabel(expanded, b.ReasoningStartedAt, b.ReasoningEndedAt, now, b.ReasoningDuration)
				lines = append(lines, styleReason.Render(strings.Repeat(" ", indent)+label))
				if expanded && b.Reasoning != "" {
					lines = append(lines, styleReason.Render(markdown.WrapText(b.Reasoning, width-indent)))
				}
			}
			if b.Content != "" {
				lines = append(lines, renderAssistantBlock(b.Content, width)...)
			} else if b.Streaming {
				lines = append(lines, renderAssistantLine(styleReason.Render("…")))
			}
			if b.TurnDuration > 0 {
				lines = append(lines, styleTurnMeta.Render(strings.Repeat(" ", indent)+turnDurationLine(b.TurnDuration)))
			}
			lines = append(lines, "")
		case RoleTool:
			lines = append(lines, chattool.Render(chattool.Block{
				Name: b.ToolName, Args: b.ToolArgs, Command: b.ToolCommand,
				Result: b.ToolResult, Running: b.ToolRunning, Error: b.ToolError,
				Expanded: b.ToolExpanded,
			}, width, showToolDetails)...)
			lines = append(lines, "")
		case RolePlanning:
			indent := lipgloss.Width(planningBullet)
			lines = append(lines, styleReason.Render(strings.Repeat(" ", indent)+planningBlockLabel(b.PlanningStartedAt, now)))
			lines = append(lines, "")
		case RoleInterrupt:
			indent := lipgloss.Width(interruptBullet)
			lines = append(lines, styleInterrupt.Render(strings.Repeat(" ", indent)+interruptBullet+interruptLabel))
			lines = append(lines, "")
		}
	}
	return strings.TrimRight(strings.Join(lines, "\n"), "\n")
}
