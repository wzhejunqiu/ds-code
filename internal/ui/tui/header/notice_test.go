package header

import (
	"strings"
	"testing"
	"unicode/utf8"

	"charm.land/lipgloss/v2"
	"github.com/wzhejunqiu/ds-code/internal/logging"
	"github.com/wzhejunqiu/ds-code/internal/mcp"
)

func TestFormatMCPSkippedSummary(t *testing.T) {
	text := FormatMCPSkippedSummary([]mcp.SkippedTool{
		{Server: "fs", Tool: "grep", Reason: mcp.SkipBuiltinConflict},
	})
	if !strings.Contains(text, "MCP 跳过") || !strings.Contains(text, "grep@fs") {
		t.Fatalf("text = %q", text)
	}
}

func TestFormatMCPSkippedSummary_allItems(t *testing.T) {
	var skipped []mcp.SkippedTool
	for i := 0; i < 5; i++ {
		skipped = append(skipped, mcp.SkippedTool{Server: "s", Tool: "t", Reason: mcp.SkipBuiltinConflict})
	}
	text := FormatMCPSkippedSummary(skipped)
	if strings.Contains(text, "另有") {
		t.Fatalf("should list all items, got %q", text)
	}
	if strings.Count(text, "t@s") != 5 {
		t.Fatalf("expected 5 detail lines, got %q", text)
	}
}

func TestWrapCells_utf8Safe(t *testing.T) {
	msg := logging.SensitiveDataWarningMsg
	lines := wrapCells(msg, 30)
	for _, line := range lines {
		if !utf8.ValidString(line) {
			t.Fatalf("invalid UTF-8: %q", line)
		}
		if lipgloss.Width(line) > 30 {
			t.Fatalf("line too wide (%d): %q", lipgloss.Width(line), line)
		}
	}
	joined := strings.Join(lines, "")
	if !strings.Contains(joined, "敏感") {
		t.Fatalf("lost content: %q", joined)
	}
}

func TestBuildNoticeLines_wrapAndPrefix(t *testing.T) {
	notices := []Notice{{Level: NoticeWarn, Text: logging.SensitiveDataWarningMsg}}
	lines := BuildNoticeLines(notices, 40)
	if len(lines) < 2 {
		t.Fatalf("expected wrap into multiple lines, got %d", len(lines))
	}
	if !strings.HasPrefix(lines[0].text, warnPrefix) {
		t.Fatalf("first line = %q", lines[0].text)
	}
	prefixWidth := lipgloss.Width(warnPrefix)
	if !strings.HasPrefix(lines[1].text, strings.Repeat(" ", prefixWidth)) {
		t.Fatalf("continuation should indent after prefix, got %q", lines[1].text)
	}
}

func TestMaxScrollOffset(t *testing.T) {
	var notices []Notice
	for i := 0; i < 3; i++ {
		notices = append(notices, Notice{Level: NoticeWarn, Text: strings.Repeat("行", 20)})
	}
	if MaxScrollOffset(notices, 20) <= 0 {
		t.Fatal("expected scrollable notices")
	}
}

func TestRenderNotificationZone_scrollHint(t *testing.T) {
	var notices []Notice
	for i := 0; i < 10; i++ {
		notices = append(notices, Notice{Level: NoticeInfo, Text: "line " + strings.Repeat("x", 30)})
	}
	out := renderNotificationZone(notices, 40, 0)
	if !strings.Contains(out, "通知 1–") {
		t.Fatalf("missing scroll counter: %q", out)
	}
	if strings.Contains(out, "Alt+") {
		t.Fatalf("should not mention keyboard shortcut: %q", out)
	}
}
