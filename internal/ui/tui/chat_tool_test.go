package tui

import (
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
	for _, want := range []string{"shell", "echo hi"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
	for _, absent := range []string{"args:", "result:", "hi\n"} {
		if strings.Contains(out, absent) {
			t.Fatalf("collapsed view should not show %q:\n%s", absent, out)
		}
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
	for _, want := range []string{"shell", "args:", "command:", "echo hi", "result:", "hi"} {
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
