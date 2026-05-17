package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func TestRenderChatUserHighlightNoLabels(t *testing.T) {
	blocks := []chatBlock{
		{role: chatRoleUser},
		{role: chatRoleAssistant},
	}
	blocks[0].content.WriteString("你好")
	blocks[1].content.WriteString("你好！")

	out := renderChat(blocks, 40, time.Now(), false)
	if strings.Contains(out, "You") || strings.Contains(out, "Assistant") {
		t.Fatalf("unexpected role labels in output:\n%s", out)
	}
	if !strings.Contains(out, userPrompt) {
		t.Fatalf("missing user prompt in output:\n%s", out)
	}
	if !strings.Contains(out, assistantBullet) {
		t.Fatalf("missing assistant bullet in output:\n%s", out)
	}
	if !strings.Contains(out, "你好") {
		t.Fatalf("missing message text in output:\n%s", out)
	}
}

func TestRenderChatTurnDuration(t *testing.T) {
	blocks := []chatBlock{{role: chatRoleAssistant}}
	blocks[0].content.WriteString("done")
	blocks[0].turnDuration = 5*time.Second + 200*time.Millisecond

	out := renderChat(blocks, 40, time.Now(), false)
	if !strings.Contains(out, "task took 5.2s") {
		t.Fatalf("expected turn duration line:\n%s", out)
	}
}

func TestRenderChatThinkingAboveContent(t *testing.T) {
	started := time.Date(2026, 5, 17, 12, 0, 0, 0, time.UTC)
	now := started.Add(12 * time.Second)

	blocks := []chatBlock{{role: chatRoleAssistant, reasoningOpen: true}}
	blocks[0].reasoning.WriteString("plan")
	blocks[0].content.WriteString("answer")
	blocks[0].reasoningStartedAt = started
	blocks[0].reasoningEndedAt = now

	out := renderChat(blocks, 40, now, false)
	thinkIdx := strings.Index(out, "thought for")
	contentIdx := strings.Index(out, "answer")
	if thinkIdx < 0 || contentIdx < 0 {
		t.Fatalf("missing thought label or content:\n%s", out)
	}
	if thinkIdx > contentIdx {
		t.Fatalf("thought label should appear above content:\n%s", out)
	}
	if !strings.Contains(out, "thought for 12s") {
		t.Fatalf("expected thought duration in output:\n%s", out)
	}
}

func TestFormatThinkingDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{500 * time.Millisecond, "0.5s"},
		{1200 * time.Millisecond, "1.2s"},
		{10 * time.Second, "10s"},
		{12 * time.Second, "12s"},
		{90 * time.Second, "1m 30s"},
	}
	for _, tc := range tests {
		if got := formatThinkingDuration(tc.d); got != tc.want {
			t.Errorf("formatThinkingDuration(%v) = %q, want %q", tc.d, got, tc.want)
		}
	}
}

func TestRenderAssistantMarkdown(t *testing.T) {
	content := "# Title\n\n**bold** and `code`"
	lines := renderAssistantBlock(content, 60)
	joined := strings.Join(lines, "\n")
	if !headingTextPresent(joined, "Title") || !strings.Contains(joined, "bold") {
		t.Fatalf("expected rendered markdown text:\n%s", joined)
	}
	if strings.Contains(joined, "**") || strings.Contains(joined, "`code`") {
		t.Fatalf("raw markdown markers in output:\n%s", joined)
	}
	if !strings.Contains(stripANSI(lines[0]), assistantBullet) {
		t.Fatalf("first line should keep assistant bullet:\n%s", joined)
	}
}

func TestRenderAssistantMarkdownHeadings(t *testing.T) {
	content := "# Title\n\n## Subtitle\n\n### Section\n\nbody"
	lines := renderAssistantBlock(content, 60)
	joined := strings.Join(lines, "\n")
	plain := stripANSI(joined)
	for _, marker := range []string{"# ", "## ", "### ", "#### "} {
		if strings.Contains(plain, marker) {
			t.Fatalf("raw heading marker %q in output:\n%s", marker, plain)
		}
	}
	for _, text := range []string{"Title", "Subtitle", "Section", "body"} {
		if !headingTextPresent(joined, text) {
			t.Fatalf("missing %q in output:\n%s", text, joined)
		}
	}
}

func TestRenderAssistantMarkdownCodeBlock(t *testing.T) {
	content := "```go\nfmt.Println(\"hi\")\n```"
	lines := renderAssistantBlock(content, 60)
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "Println") || !strings.Contains(joined, "hi") {
		t.Fatalf("expected code block content:\n%s", joined)
	}
}

func TestRenderChatUserFullWidthBackground(t *testing.T) {
	blocks := []chatBlock{{role: chatRoleUser}}
	blocks[0].content.WriteString("hi")

	out := renderChat(blocks, 30, time.Now(), false)
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 1 {
		t.Fatalf("got %d lines, want 1:\n%s", len(lines), out)
	}
	if lipgloss.Width(lines[0]) != 30 {
		t.Fatalf("user line width = %d, want 30:\n%q", lipgloss.Width(lines[0]), lines[0])
	}
}
