package chattool

import (
	"fmt"

	"github.com/wzhejunqiu/ds-code/internal/tool"
)

// Line renders a compact one-line tool summary for the side panel.
func Line(name, args, command, preview string, running, isError bool) string {
	label := sidebarLabel(name, args, command)
	var s string
	switch {
	case running:
		s = fmt.Sprintf("→ %s …", label)
	case isError:
		s = fmt.Sprintf("✗ %s", label)
	default:
		s = fmt.Sprintf("✓ %s", label)
	}
	if !tool.UsesHumanDisplay(name) && !tool.IsShellDisplay(name) && !tool.IsApplyPatchDisplay(name) {
		if command != "" {
			s += "  " + truncate(command, 60)
		} else if args != "" {
			s += "  " + truncate(args, 60)
		}
	}
	if !running && preview != "" && preview != "…" {
		s += "  " + truncate(preview, 80)
	}
	if isError {
		return styleToolError.Render(s)
	}
	return styleTool.Render(s)
}

func sidebarLabel(name, args, command string) string {
	if human := tool.HumanToolTitle(name, args, command); human != "" {
		return human
	}
	if tool.IsShellDisplay(name) && args != "" {
		cmds := tool.ShellCommandsList(command)
		if cmds != "" {
			return args + " " + cmds
		}
		return args
	}
	if tool.IsApplyPatchDisplay(name) && args != "" {
		added, removed, _ := tool.DecodeApplyPatchStats(command)
		line := "Edit " + args
		if added > 0 {
			line += fmt.Sprintf(" +%d", added)
		}
		if removed > 0 {
			line += fmt.Sprintf(" -%d", removed)
		}
		return line
	}
	return name
}
