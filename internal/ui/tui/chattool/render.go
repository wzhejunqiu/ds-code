package chattool

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/wzhejunqiu/ds-code/internal/tool"
)

const (
	bullet             = "⚙ "
	resultConnector    = "└ "
	resultMax          = 2000
	resultPreviewMax   = 256
	resultPreviewLines = 3
	titleArgsMax       = 80
)

// Render returns styled lines for a tool block in the main chat transcript.
func Render(b Block, width int, showDetails bool) []string {
	expanded := showDetails || b.Expanded
	return renderBlock(b, width, expanded)
}

func renderBlock(b Block, width int, expanded bool) []string {
	indent := lipgloss.Width(bullet)
	var body []string

	if b.Running {
		body = append(body, renderRunningTitle(b.Name, b.Command, b.Args))
		if !tool.UsesHumanDisplay(b.Name) && !tool.IsShellDisplay(b.Name) && !tool.IsApplyPatchDisplay(b.Name) {
			switch {
			case b.Command != "":
				body = append(body, styleToolMeta.Render(strings.Repeat(" ", indent)+truncate(b.Command, titleArgsMax)))
			case b.Args != "":
				body = append(body, styleToolMeta.Render(strings.Repeat(" ", indent)+truncate(b.Args, titleArgsMax)))
			default:
				body = append(body, styleToolMeta.Render(strings.Repeat(" ", indent)+"running…"))
			}
		} else if toolRunningTitle(b.Name, b.Args, b.Command) == "" && !tool.IsShellDisplay(b.Name) && !tool.IsApplyPatchDisplay(b.Name) {
			body = append(body, styleToolMeta.Render(strings.Repeat(" ", indent)+"running…"))
		}
		return body
	}

	body = append(body, renderTitleLine(b.Name, b.Command, b.Args, b.Error))

	if expanded {
		if b.Args != "" && !skipExpandedArgs(b.Name) {
			body = append(body, styleToolMeta.Render(strings.Repeat(" ", indent)+"args: "+b.Args))
		}
		if b.Command != "" && tool.IsShellDisplay(b.Name) {
			full := tool.ShellFullCommand(b.Command)
			body = append(body, styleToolMeta.Render(strings.Repeat(" ", indent)+"command: "+full))
		} else if b.Command != "" && !tool.IsApplyPatchDisplay(b.Name) {
			body = append(body, styleToolMeta.Render(strings.Repeat(" ", indent)+"command: "+b.Command))
		}
		if b.Result != "" {
			body = append(body, renderResultLines(truncate(b.Result, resultMax), indent, width, b.Error)...)
		}
		return body
	}

	body = append(body, renderResultCollapsed(b.Result, indent, width, b.Error)...)
	return body
}

func skipExpandedArgs(name string) bool {
	return tool.UsesHumanDisplay(name) || tool.IsShellDisplay(name) || tool.IsApplyPatchDisplay(name)
}

func renderTitleLine(name, command, args string, isError bool) string {
	if tool.IsShellDisplay(name) {
		return renderShellTitle(args, command, isError)
	}
	if tool.IsApplyPatchDisplay(name) {
		return renderApplyPatchTitle(args, command, isError)
	}
	if human := tool.HumanToolTitle(name, args, command); human != "" {
		parts := []string{styleToolName.Render(bullet + human)}
		if isError {
			parts = append(parts, styleToolError.Render(" (error)"))
		}
		return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
	}
	parts := []string{styleToolName.Render(bullet + name)}
	if paren := parenContent(command, args); paren != "" {
		parts = append(parts, styleToolCommand.Render(" ("+paren+")"))
	}
	if isError {
		parts = append(parts, styleToolError.Render(" (error)"))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

func renderShellTitle(description, commandField string, isError bool) string {
	commands := tool.ShellCommandsList(commandField)
	parts := []string{styleToolName.Render(bullet + description)}
	if commands != "" {
		parts = append(parts, styleToolShellCmds.Render(" "+commands))
	}
	if isError {
		parts = append(parts, styleToolError.Render(" (error)"))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

func renderApplyPatchTitle(filename, statsEnc string, isError bool) string {
	added, removed, _ := tool.DecodeApplyPatchStats(statsEnc)
	parts := []string{styleToolName.Render(bullet + "Edit " + filename)}
	if added > 0 {
		parts = append(parts, styleToolSuccess.Render(fmt.Sprintf(" +%d", added)))
	}
	if removed > 0 {
		parts = append(parts, styleToolError.Render(fmt.Sprintf(" -%d", removed)))
	}
	if isError {
		parts = append(parts, styleToolError.Render(" (error)"))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

func renderRunningTitle(name, command, args string) string {
	line := renderTitleLine(name, command, args, false)
	return lipgloss.JoinHorizontal(lipgloss.Top, line, styleToolMeta.Render(" …"))
}

func toolRunningTitle(name, args, command string) string {
	if human := tool.HumanToolTitle(name, args, command); human != "" {
		return human
	}
	if tool.IsShellDisplay(name) {
		return args
	}
	if tool.IsApplyPatchDisplay(name) {
		return "Edit " + args
	}
	return name
}

func parenContent(command, args string) string {
	paren := command
	if paren == "" {
		paren = args
		if len(paren) > titleArgsMax {
			paren = truncate(paren, titleArgsMax)
		}
	}
	return paren
}

type resultPreviewData struct {
	lines     []string
	moreLines int
	truncated bool
}

func buildResultPreview(result string) resultPreviewData {
	if result == "" {
		return resultPreviewData{}
	}
	totalLines := countResultLines(result)
	truncated := len(result) > resultPreviewMax
	s := result
	if truncated {
		s = truncateTo(result, resultPreviewMax)
	}
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	shown := len(lines)
	if shown > resultPreviewLines {
		lines = lines[:resultPreviewLines]
		shown = resultPreviewLines
	}
	moreLines := totalLines - shown
	if moreLines < 0 {
		moreLines = 0
	}
	return resultPreviewData{lines: lines, moreLines: moreLines, truncated: truncated}
}

func countResultLines(s string) int {
	if s == "" {
		return 0
	}
	return len(strings.Split(strings.TrimRight(s, "\n"), "\n"))
}

func expandHint(moreLines int, truncated bool) string {
	switch {
	case moreLines > 0:
		return fmt.Sprintf("... +%d lines (ctrl+o to expand)", moreLines)
	case truncated:
		return "... (ctrl+o to expand)"
	default:
		return ""
	}
}

func renderResultCollapsed(result string, indent, width int, isError bool) []string {
	preview := buildResultPreview(result)
	if len(preview.lines) == 0 {
		return nil
	}
	lines := renderResultLines(strings.Join(preview.lines, "\n"), indent, width, isError)
	if hint := expandHint(preview.moreLines, preview.truncated); hint != "" {
		connWidth := lipgloss.Width(resultConnector)
		lines = append(lines, styleToolExpandHint.Render(strings.Repeat(" ", indent+connWidth)+hint))
	}
	return lines
}

func renderResultLines(result string, indent, width int, isError bool) []string {
	if result == "" {
		return nil
	}
	style := styleToolResult
	if isError {
		style = styleToolError
	}
	text := strings.TrimRight(result, "\n")
	if width > 0 {
		connWidth := lipgloss.Width(resultConnector)
		text = wrapText(text, width-indent-connWidth)
	}
	raw := strings.Split(text, "\n")
	connWidth := lipgloss.Width(resultConnector)
	out := make([]string, 0, len(raw))
	for i, line := range raw {
		pad := strings.Repeat(" ", indent)
		if i == 0 {
			out = append(out, style.Render(pad+resultConnector+line))
		} else {
			out = append(out, style.Render(pad+strings.Repeat(" ", connWidth)+line))
		}
	}
	return out
}
