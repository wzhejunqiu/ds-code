package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestRenderChatToolBlockCollapsed(t *testing.T) {
	blocks := []chatBlock{{
		role:        chatRoleTool,
		toolName:    "shell",
		toolArgs:    `{"command":"echo hi"}`,
		toolCommand: "echo hi",
		toolResult:  "hi\n",
	}}
	out := renderChat(blocks, 60, time.Now(), false)
	for _, want := range []string{"shell", "(echo hi)", "└", "hi"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
	for _, absent := range []string{"args:", "result:", "ctrl+o"} {
		if strings.Contains(out, absent) {
			t.Fatalf("collapsed view should not show %q:\n%s", absent, out)
		}
	}
}

func TestBuildToolResultPreview(t *testing.T) {
	got := buildToolResultPreview("one\ntwo\nthree\nfour")
	if len(got.lines) != 3 || got.lines[0] != "one" || got.moreLines != 1 {
		t.Fatalf("preview = %+v", got)
	}
	longLine := strings.Repeat("x", 300)
	got = buildToolResultPreview(longLine)
	if len(got.lines) != 1 || len(got.lines[0]) > toolResultPreviewMax {
		t.Fatalf("preview = %+v", got)
	}
	if !got.truncated || !strings.HasSuffix(got.lines[0], "...") {
		t.Fatalf("long line should be truncated: %+v", got)
	}
}

func TestRenderChatToolBlockExpandHint(t *testing.T) {
	var lines []string
	for i := 1; i <= 6; i++ {
		lines = append(lines, fmt.Sprintf("line%d", i))
	}
	blocks := []chatBlock{{
		role:        chatRoleTool,
		toolName:    "shell",
		toolCommand: "seq",
		toolResult:  strings.Join(lines, "\n"),
	}}
	out := renderChat(blocks, 60, time.Now(), false)
	if !strings.Contains(out, "+3 lines (ctrl+o to expand)") {
		t.Fatalf("missing expand hint:\n%s", out)
	}
}

func TestRenderChatToolBlockExpanded(t *testing.T) {
	blocks := []chatBlock{{
		role:        chatRoleTool,
		toolName:    "shell",
		toolArgs:    `{"command":"echo hi"}`,
		toolCommand: "echo hi",
		toolResult:  "hi\n",
	}}
	out := renderChat(blocks, 60, time.Now(), true)
	for _, want := range []string{"shell", "(echo hi)", "args:", "command:", "echo hi", "└", "hi"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}

func TestRenderChatToolBeforeAssistant(t *testing.T) {
	blocks := []chatBlock{
		{role: chatRoleTool, toolName: "read_file", toolArgs: "path=main.go", toolResult: "package main"},
		{role: chatRoleAssistant},
	}
	blocks[1].content.WriteString("done")

	out := renderChat(blocks, 50, time.Now(), false)
	toolIdx := strings.Index(out, "read_file")
	contentIdx := strings.Index(out, "done")
	if toolIdx < 0 || contentIdx < 0 || toolIdx > contentIdx {
		t.Fatalf("tool should appear before assistant content:\n%s", out)
	}
}
