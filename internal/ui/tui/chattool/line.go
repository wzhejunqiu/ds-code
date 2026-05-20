package chattool

import (
	"fmt"

	"github.com/hejunqiu/ds-code/internal/tool"
)

// Line renders a compact one-line tool summary for the side panel.
func Line(name, args, command, preview string, running, isError bool) string {
	label := name
	if human := tool.HumanToolTitle(name, args, command); human != "" {
		label = human
	}
	var s string
	switch {
	case running:
		s = fmt.Sprintf("→ %s …", label)
	case isError:
		s = fmt.Sprintf("✗ %s", label)
	default:
		s = fmt.Sprintf("✓ %s", label)
	}
	if human := tool.HumanToolTitle(name, args, command); human == "" {
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
