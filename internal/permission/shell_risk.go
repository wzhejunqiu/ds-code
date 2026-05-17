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
	"|sh",
	"|bash",
	"> /dev/sd",
	"dd if=",
	"chmod 777 /",
	"chmod -r 777",
}

func normalizeShellCmd(cmd string) string {
	lower := strings.ToLower(cmd)
	var b strings.Builder
	b.Grow(len(lower) + 16)
	for i := 0; i < len(lower); i++ {
		r := lower[i]
		switch r {
		case ' ', '\t', '\n', '\r':
			if b.Len() > 0 && b.String()[b.Len()-1] != ' ' {
				b.WriteByte(' ')
			}
		case '|', '&', ';':
			if b.Len() > 0 && b.String()[b.Len()-1] != ' ' {
				b.WriteByte(' ')
			}
			b.WriteByte(r)
			if i+1 < len(lower) && lower[i+1] != ' ' {
				b.WriteByte(' ')
			}
		default:
			b.WriteByte(r)
		}
	}
	return strings.TrimSpace(b.String())
}
