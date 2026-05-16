package slash

import (
	"regexp"
	"strings"
)

var parseRE = regexp.MustCompile(`^/([a-z][a-z0-9_-]*)(?:\s+(.*))?$`)

// Parse recognizes a slash command when the trimmed line starts with /
// and matches the canonical pattern. Line-internal /path is not a command.
func Parse(line string) (cmd, args string, ok bool) {
	line = strings.TrimSpace(line)
	if line == "" || line[0] != '/' {
		return "", "", false
	}
	m := parseRE.FindStringSubmatch(line)
	if m == nil {
		return "", "", false
	}
	args = ""
	if len(m) > 2 {
		args = m[2]
	}
	return m[1], args, true
}
