package header

import (
	"fmt"
	"strings"

	"github.com/wzhejunqiu/ds-code/internal/mcp"
)

// Level is the visual severity of a startup notice.
type Level int

const (
	NoticeInfo Level = iota
	NoticeWarn
)

// Notice is one startup message shown in the TUI header.
type Notice struct {
	Level Level
	Text  string
}

var skipReasonLabel = map[mcp.SkipReason]string{
	mcp.SkipBuiltinConflict:      "内建冲突",
	mcp.SkipCrossServerDuplicate: "跨 server 重复",
	mcp.SkipInServerDuplicate:    "server 内重复",
}

// FormatMCPSkippedSummary aggregates skipped MCP tools for the header.
func FormatMCPSkippedSummary(skipped []mcp.SkippedTool) string {
	if len(skipped) == 0 {
		return ""
	}
	lines := []string{fmt.Sprintf("MCP 跳过 %d 个工具", len(skipped))}
	for _, s := range skipped {
		reason := string(s.Reason)
		if label, ok := skipReasonLabel[s.Reason]; ok {
			reason = label
		}
		lines = append(lines, fmt.Sprintf("%s@%s (%s)", s.Tool, s.Server, reason))
	}
	return strings.Join(lines, "\n")
}
