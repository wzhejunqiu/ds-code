package permission

import "strings"

var highRiskShellPatterns = []string{
	"rm -rf /",
	"rm -rf ~",
	"rm -rf $home",
	"mkfs",
	":(){",
	"curl | sh",
	"curl | bash",
	"wget | sh",
	"wget | bash",
	"| sh",
	"| bash",
	"> /dev/sd",
	"dd if=",
	"chmod 777 /",
	"chmod -r 777",
}

func normalizeShellCmd(cmd string) string {
	var b strings.Builder
	b.Grow(len(cmd))
	prevSpace := false
	for _, r := range strings.ToLower(cmd) {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if !prevSpace && b.Len() > 0 {
				b.WriteByte(' ')
				prevSpace = true
			}
			continue
		}
		b.WriteRune(r)
		prevSpace = false
	}
	return strings.TrimSpace(b.String())
}
