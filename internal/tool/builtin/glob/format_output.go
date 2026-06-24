package glob

import (
	"strings"

	"github.com/wzhejunqiu/ds-code/internal/tool/builtin/rgutil"
)

func formatGlobOutput(paths []string, total, limit int) string {
	if total == 0 {
		return rgutil.FormatFilesSummary(0)
	}
	var b strings.Builder
	b.WriteString(rgutil.FormatFilesSummary(len(paths)))
	for _, p := range paths {
		b.WriteByte('\n')
		b.WriteString(p)
	}
	if limit > 0 && total > limit {
		b.WriteByte('\n')
		b.WriteString(MsgResultsTruncated)
	}
	return b.String()
}
