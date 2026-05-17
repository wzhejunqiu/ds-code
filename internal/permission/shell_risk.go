package permission

import (
	"fmt"
	"strings"
)

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
	"|sh",
	"|bash",
	"> /dev/sd",
	"dd if=",
	"chmod 777 /",
	"chmod -r 777",
	"base64 -d",
	"base64 --decode",
	"openssl enc",
	"> /etc/",
	"tee /etc/",
	"chmod +s",
	"chown root",
}

var highRiskShellPrefixes = []string{
	"sudo ",
	"doas ",
	"eval ",
	"exec ",
}

func matchHighRiskShell(norm string) error {
	for _, p := range highRiskShellPrefixes {
		if strings.HasPrefix(norm, p) {
			return fmt.Errorf("%w: high-risk shell command blocked", ErrDenied)
		}
	}
	for _, p := range highRiskShellPatterns {
		if strings.Contains(norm, p) {
			return fmt.Errorf("%w: high-risk shell command blocked", ErrDenied)
		}
	}
	return nil
}

func normalizeShellCmd(cmd string) string {
	lower := strings.ToLower(cmd)
	var b strings.Builder
	b.Grow(len(lower) + 16)
	lastSpace := false
	for i := 0; i < len(lower); i++ {
		r := lower[i]
		switch r {
		case ' ', '\t', '\n', '\r':
			if b.Len() > 0 && !lastSpace {
				b.WriteByte(' ')
				lastSpace = true
			}
		case '|', '&', ';':
			if b.Len() > 0 && !lastSpace {
				b.WriteByte(' ')
			}
			b.WriteByte(r)
			lastSpace = false
			if i+1 < len(lower) && lower[i+1] != ' ' {
				b.WriteByte(' ')
				lastSpace = true
			}
		default:
			b.WriteByte(r)
			lastSpace = false
		}
	}
	return strings.TrimSpace(b.String())
}
