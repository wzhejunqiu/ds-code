package tool

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hejunqiu/ds-code/internal/patch"
)

// DisplaySummary formats tool arguments for the TUI (args line and optional command).
func DisplaySummary(name string, rawArgs []byte, workspace string) (argsLine, command string) {
	args := ArgsMap(rawArgs)
	switch name {
	case "shell":
		if c, _ := args["command"].(string); c != "" {
			return formatArgsJSON(rawArgs), c
		}
	case "write_file":
		if p, _ := args["path"].(string); p != "" {
			return "path=" + p, ""
		}
	case "read_file":
		if p, _ := args["path"].(string); p != "" {
			line := "path=" + p
			if st, ok := args["start"].(float64); ok && st > 0 {
				line += fmt.Sprintf(", start=%d", int(st))
			}
			if en, ok := args["end"].(float64); ok && en > 0 {
				line += fmt.Sprintf(", end=%d", int(en))
			}
			return line, ""
		}
	case "apply_patch":
		if p, _ := args["patch"].(string); p != "" {
			paths, err := patch.Paths(p, workspace)
			if err == nil {
				return "files: " + strings.Join(paths, ", "), ""
			}
			return "patch", ""
		}
	case "task":
		if d, _ := args["description"].(string); d != "" {
			return d, ""
		}
		if p, _ := args["prompt"].(string); p != "" {
			return truncateOneLine(p, 120), ""
		}
	default:
		if strings.HasPrefix(name, "mcp__") {
			return formatArgsJSON(rawArgs), ""
		}
	}
	if len(rawArgs) > 0 {
		return formatArgsJSON(rawArgs), ""
	}
	return "", ""
}

func truncateOneLine(s string, max int) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func formatArgsJSON(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	var compact any
	if err := json.Unmarshal(raw, &compact); err != nil {
		s := strings.TrimSpace(string(raw))
		if len(s) > 400 {
			return s[:400] + "..."
		}
		return s
	}
	b, err := json.Marshal(compact)
	if err != nil {
		return string(raw)
	}
	s := string(b)
	if len(s) > 400 {
		return s[:400] + "..."
	}
	return s
}
