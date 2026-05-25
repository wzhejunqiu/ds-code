// Package classifier applies rule-based shell command decisions before execution.
package classifier

import (
	"regexp"
	"strings"
)

// Decision is the outcome of classifying a shell command.
type Decision string

const (
	Allow Decision = "allow"
	Deny  Decision = "deny"
	Ask   Decision = "ask"
)

var (
	reRmRF        = regexp.MustCompile(`(?i)\brm\s+(-[a-zA-Z]*r[a-zA-Z]*f|-[a-zA-Z]*f[a-zA-Z]*r|\s+-rf\b|\s+rf\s|-rf\s)`)
	reCurlPipe    = regexp.MustCompile(`(?i)(curl|wget).*\|\s*(ba)?sh`)
	reChmod777    = regexp.MustCompile(`(?i)\bchmod\s+777\b`)
	reGitPush     = regexp.MustCompile(`(?i)^git\s+push\b`)
	reDocker      = regexp.MustCompile(`(?i)\bdocker\b`)
	reSudo        = regexp.MustCompile(`(?i)\bsudo\b`)
	reGitReadOnly = regexp.MustCompile(`(?i)^git\s+(status|diff|log|show|branch|rev-parse)\b`)
	reShellWrap   = regexp.MustCompile(`(?i)(?:^|;\s*|\|\|\s*|\&\&\s*)(?:/usr/bin/)?(?:env\s+)?(?:bash|sh|zsh|dash)\s+(?:-[a-zA-Z]+\s+)*-c\s+`)
)

// Classify returns a decision and human-readable reason for a shell command string.
func Classify(cmd string) (Decision, string) {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return Allow, ""
	}
	best := Allow
	reason := ""
	for _, part := range unwrapShell(cmd) {
		dec, r := classifyOne(part)
		if severity(dec) > severity(best) {
			best = dec
			reason = r
		}
	}
	return best, reason
}

func classifyOne(cmd string) (Decision, string) {
	lower := strings.ToLower(strings.TrimSpace(cmd))
	if lower == "" {
		return Allow, ""
	}
	if reRmRF.MatchString(lower) || reCurlPipe.MatchString(lower) || reChmod777.MatchString(lower) {
		return Deny, "high-risk shell pattern"
	}
	if reSudo.MatchString(lower) || reDocker.MatchString(lower) || reGitPush.MatchString(lower) {
		return Ask, "privileged or remote-changing command"
	}
	if reGitReadOnly.MatchString(lower) {
		return Allow, "read-only git"
	}
	return Allow, ""
}

func severity(d Decision) int {
	switch d {
	case Deny:
		return 3
	case Ask:
		return 2
	default:
		return 1
	}
}

// unwrapShell returns the outer command plus any inner commands from shell -c wrappers.
func unwrapShell(cmd string) []string {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return nil
	}
	out := []string{cmd}
	rest := cmd
	for {
		loc := reShellWrap.FindStringIndex(rest)
		if loc == nil {
			break
		}
		innerStart := loc[1]
		if innerStart >= len(rest) {
			break
		}
		inner, consumed := parseQuotedShellArg(rest[innerStart:])
		if inner != "" {
			out = append(out, inner)
		}
		if consumed <= 0 {
			break
		}
		rest = strings.TrimSpace(rest[innerStart+consumed:])
	}
	return out
}

func parseQuotedShellArg(s string) (arg string, consumed int) {
	s = strings.TrimLeft(s, " \t")
	if s == "" {
		return "", 0
	}
	switch s[0] {
	case '\'':
		end := strings.Index(s[1:], "'")
		if end < 0 {
			return s, len(s)
		}
		return s[1 : 1+end], 1 + end + 1
	case '"':
		end := strings.Index(s[1:], "\"")
		if end < 0 {
			return s, len(s)
		}
		return s[1 : 1+end], 1 + end + 1
	default:
		fields := strings.Fields(s)
		if len(fields) == 0 {
			return "", 0
		}
		first := fields[0]
		return first, strings.Index(s, first) + len(first)
	}
}
