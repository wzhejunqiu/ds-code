package chattool

import (
	"fmt"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/tool"
)

// Line renders a compact one-line tool summary for the side panel.
func Line(name, args, command, preview string, running, isError bool, deadline time.Time, now time.Time, disp tool.DisplayContext) string {
	label := sidebarLabel(name, args, command, disp)
	var s string
	switch {
	case running:
		s = fmt.Sprintf("→ %s …", label)
		if tool.IsShellDisplay(name) && !deadline.IsZero() {
			if cd := tool.FormatTimeoutCountdown(deadline, now); cd != "" {
				s = fmt.Sprintf("→ %s  %s", label, cd)
			}
		}
	case isError:
		s = fmt.Sprintf("✗ %s", label)
	default:
		s = fmt.Sprintf("✓ %s", label)
	}
	if !tool.UsesHumanDisplay(name, disp) && !tool.IsShellDisplay(name) && !tool.IsApplyPatchDisplay(name) {
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

func sidebarLabel(name, args, command string, disp tool.DisplayContext) string {
	if human := tool.HumanToolTitle(name, args, command, disp); human != "" {
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
