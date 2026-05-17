package chattool

import "fmt"

// Line renders a compact one-line tool summary for the side panel.
func Line(name, args, command, preview string, running, isError bool) string {
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
		return styleToolError.Render(s)
	}
	return styleTool.Render(s)
}
