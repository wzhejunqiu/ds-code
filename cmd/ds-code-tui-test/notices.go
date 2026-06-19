//go:build tuitest

package main

import (
	"fmt"
	"strings"

	"github.com/wzhejunqiu/ds-code/internal/logging"
	"github.com/wzhejunqiu/ds-code/internal/mcp"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/header"
)

// harnessStartupNotices returns demo notices for the interactive TUI harness.
// Total wrapped lines exceed maxNoticeVisibleLines so the notification zone auto-scrolls.
func harnessStartupNotices() []header.Notice {
	skipped := []mcp.SkippedTool{
		{Server: "code-review-graph", Tool: "semantic_search_nodes", Reason: mcp.SkipCrossServerDuplicate},
		{Server: "fs", Tool: "grep", Reason: mcp.SkipBuiltinConflict},
		{Server: "fs", Tool: "glob", Reason: mcp.SkipBuiltinConflict},
		{Server: "cursor", Tool: "read_file", Reason: mcp.SkipCrossServerDuplicate},
		{Server: "mcp-a", Tool: "search", Reason: mcp.SkipInServerDuplicate},
		{Server: "mcp-b", Tool: "search", Reason: mcp.SkipCrossServerDuplicate},
	}
	notices := []header.Notice{
		{Level: header.NoticeWarn, Text: logging.SensitiveDataWarningMsg},
		{Level: header.NoticeWarn, Text: header.FormatMCPSkippedSummary(skipped)},
	}
	for i := 1; i <= 4; i++ {
		notices = append(notices, header.Notice{
			Level: header.NoticeInfo,
			Text: fmt.Sprintf(
				"Harness 演示通知 %d/4：header 右侧消息通知区在内容超出可见行数时每约 4 秒自动滚动，无需快捷键。%s",
				i, strings.Repeat("·", 24),
			),
		})
	}
	return notices
}
