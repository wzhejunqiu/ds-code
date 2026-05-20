package tool

import (
	"encoding/json"
	"fmt"
	"strconv"
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
			return FormatReadFileDisplay(p, 0, 0), ""
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

// ReadFileLineRange parses "N|..." lines from read_file tool output.
func ReadFileLineRange(result string) (start, end int, ok bool) {
	for _, line := range strings.Split(result, "\n") {
		i := strings.IndexByte(line, '|')
		if i <= 0 {
			continue
		}
		n, err := strconv.Atoi(line[:i])
		if err != nil || n <= 0 {
			continue
		}
		if !ok {
			start, end = n, n
			ok = true
			continue
		}
		if n < start {
			start = n
		}
		if n > end {
			end = n
		}
	}
	return start, end, ok
}

// FormatReadFileDisplay formats a human-readable read_file label for the TUI.
func FormatReadFileDisplay(path string, start, end int) string {
	line := "Read " + path
	if start > 0 && end >= start {
		line += fmt.Sprintf(" L%d-%d", start, end)
	}
	return line
}

// AppendReadFileLineRange updates a read_file display line with the actual line range.
func AppendReadFileLineRange(argsLine string, start, end int) string {
	path := readFilePathFromDisplay(argsLine)
	if path == "" {
		return argsLine
	}
	return FormatReadFileDisplay(path, start, end)
}

func readFilePathFromDisplay(line string) string {
	line = strings.TrimSpace(line)
	if strings.HasPrefix(line, "Read ") {
		rest := strings.TrimPrefix(line, "Read ")
		if i := strings.Index(rest, " L"); i >= 0 {
			rest = rest[:i]
		}
		return strings.TrimSpace(rest)
	}
	if strings.HasPrefix(line, "path=") {
		return strings.TrimPrefix(line, "path=")
	}
	return ""
}

// HumanToolTitle returns a single-line TUI label when the tool uses a non-function style.
// Empty means use the default "tool_name (args)" rendering.
func HumanToolTitle(name, args, command string) string {
	if name == "read_file" && args != "" {
		return args
	}
	return ""
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
