package tool

import (
	"path/filepath"
	"strings"
)

// ParseShellCommands splits a shell command into executable names (cd, go, head, …).
func ParseShellCommands(cmd string) []string {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return nil
	}
	var names []string
	seen := make(map[string]struct{})
	for _, seg := range splitShellSegments(cmd) {
		name := shellExecutableName(seg)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	return names
}

// FormatShellCommandsList formats command names for the TUI (with N+ truncation).
func FormatShellCommandsList(names []string) string {
	n := len(names)
	switch {
	case n == 0:
		return ""
	case n <= 2:
		return strings.Join(names, ", ")
	case n == 3:
		return names[0] + ", 2+"
	default:
		return names[0] + ", " + itoa(n-1) + "+"
	}
}

func splitShellSegments(cmd string) []string {
	var segs []string
	var cur strings.Builder
	inSingle := false
	inDouble := false
	escaped := false
	for _, r := range cmd {
		if escaped {
			cur.WriteRune(r)
			escaped = false
			continue
		}
		if r == '\\' && (inSingle || inDouble) {
			escaped = true
			cur.WriteRune(r)
			continue
		}
		if r == '\'' && !inDouble {
			inSingle = !inSingle
			cur.WriteRune(r)
			continue
		}
		if r == '"' && !inSingle {
			inDouble = !inDouble
			cur.WriteRune(r)
			continue
		}
		if !inSingle && !inDouble && (r == '&' || r == '|' || r == ';') {
			if cur.Len() > 0 {
				segs = append(segs, cur.String())
				cur.Reset()
			}
			if r == '&' {
				// swallow && or ||
				continue
			}
			continue
		}
		cur.WriteRune(r)
	}
	if cur.Len() > 0 {
		segs = append(segs, cur.String())
	}
	// Re-split on && and || inside segments that weren't caught (no spaces).
	var out []string
	for _, s := range segs {
		out = append(out, splitAndOr(s)...)
	}
	return out
}

func splitAndOr(s string) []string {
	var parts []string
	for {
		i := indexMultiOp(s)
		if i < 0 {
			if strings.TrimSpace(s) != "" {
				parts = append(parts, s)
			}
			break
		}
		if i > 0 {
			parts = append(parts, s[:i])
		}
		s = s[i:]
		for len(s) > 0 && (s[0] == '&' || s[0] == '|') {
			s = s[1:]
		}
	}
	return parts
}

func indexMultiOp(s string) int {
	for i := 0; i < len(s)-1; i++ {
		if s[i] == '&' && s[i+1] == '&' {
			return i
		}
		if s[i] == '|' && s[i+1] == '|' {
			return i
		}
	}
	return -1
}

func shellExecutableName(seg string) string {
	seg = strings.TrimSpace(seg)
	if seg == "" {
		return ""
	}
	// Skip env assignments FOO=bar
	if i := strings.Index(seg, "="); i > 0 && !strings.Contains(seg[:i], " ") {
		return ""
	}
	fields := strings.Fields(seg)
	if len(fields) == 0 {
		return ""
	}
	tok := fields[0]
	if strings.Contains(tok, "/") {
		return filepath.Base(tok)
	}
	return tok
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [16]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// IsShellDisplay reports tools using description + commands dual-line title.
func IsShellDisplay(name string) bool {
	return NameShell.Matches(name)
}

// ShellSummary formats shell tool display (description + command).
func ShellSummary(rawArgs []byte) (argsLine, command string) {
	args := ArgsMap(rawArgs)
	desc, _ := args["description"].(string)
	desc = strings.TrimSpace(desc)
	cmd, _ := args["command"].(string)
	cmd = strings.TrimSpace(cmd)
	if desc == "" {
		if cmd != "" {
			desc = truncateOneLine(cmd, 80)
		} else {
			desc = NameShell.String()
		}
	}
	list := FormatShellCommandsList(ParseShellCommands(cmd))
	return desc, encodeShellCommandField(list, cmd)
}

const shellCommandSep = "\x00"

func encodeShellCommandField(list, fullCmd string) string {
	if list == "" {
		return fullCmd
	}
	if fullCmd == "" {
		return list
	}
	return list + shellCommandSep + fullCmd
}

// ShellCommandsList returns the TUI commands fragment (before optional full command).
func ShellCommandsList(commandField string) string {
	if i := strings.Index(commandField, shellCommandSep); i >= 0 {
		return commandField[:i]
	}
	return commandField
}

// ShellFullCommand returns the full shell command for expanded details.
func ShellFullCommand(commandField string) string {
	if i := strings.Index(commandField, shellCommandSep); i >= 0 {
		return commandField[i+len(shellCommandSep):]
	}
	return commandField
}
