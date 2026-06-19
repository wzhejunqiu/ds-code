package chat

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/wzhejunqiu/ds-code/internal/tool"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/markdown"
)

func TestRenderUserHighlightNoLabels(t *testing.T) {
	blocks := []Block{
		{Role: RoleUser},
		{Role: RoleAssistant},
	}
	blocks[0].Content = "你好"
	blocks[1].Content = "你好！"

	out := Render(blocks, 40, time.Now(), false, tool.DisplayContext{})
	if strings.Contains(out, "You") || strings.Contains(out, "Assistant") {
		t.Fatalf("unexpected role labels in output:\n%s", out)
	}
	if !strings.Contains(out, UserPrompt) {
		t.Fatalf("missing user prompt in output:\n%s", out)
	}
	if !strings.Contains(out, AssistantBullet) {
		t.Fatalf("missing assistant bullet in output:\n%s", out)
	}
}

func TestRenderReasoningExpandedWhileThinking(t *testing.T) {
	started := time.Now().Add(-500 * time.Millisecond)
	blocks := []Block{{
		Role:               RoleAssistant,
		ReasoningOpen:      false,
		ReasoningStartedAt: started,
		Streaming:          true,
	}}
	blocks[0].Reasoning = "think step"

	out := Render(blocks, 60, time.Now(), false, tool.DisplayContext{})
	if !strings.Contains(out, "think step") {
		t.Fatalf("expected reasoning body while thinking:\n%s", out)
	}
	if !strings.Contains(out, "▾") {
		t.Fatalf("expected expanded thinking label:\n%s", out)
	}
}

func TestRenderReasoningCollapsedAfterThinking(t *testing.T) {
	ended := time.Now().Add(-1 * time.Second)
	started := ended.Add(-2 * time.Second)
	blocks := []Block{{
		Role:               RoleAssistant,
		ReasoningOpen:      false,
		ReasoningStartedAt: started,
		ReasoningEndedAt:   ended,
	}}
	blocks[0].Reasoning = "think step"
	blocks[0].Content = "answer"

	out := Render(blocks, 60, time.Now(), false, tool.DisplayContext{})
	if strings.Contains(out, "think step") {
		t.Fatalf("expected reasoning body hidden after thinking:\n%s", out)
	}
	if !strings.Contains(out, "thought for") {
		t.Fatalf("expected collapsed thought label:\n%s", out)
	}
}

func TestRenderTurnDuration(t *testing.T) {
	blocks := []Block{{Role: RoleAssistant}}
	blocks[0].Content = "done"
	blocks[0].TurnDuration = 5*time.Second + 200*time.Millisecond

	out := Render(blocks, 40, time.Now(), false, tool.DisplayContext{})
	if !strings.Contains(out, "task took 5.2s") {
		t.Fatalf("expected turn duration line:\n%s", out)
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
		{90 * time.Second, "1m 30s"},
	}
	for _, tc := range tests {
		if got := FormatThinkingDuration(tc.d); got != tc.want {
			t.Errorf("FormatThinkingDuration(%v) = %q, want %q", tc.d, got, tc.want)
		}
	}
}

func TestRenderAssistantMarkdown(t *testing.T) {
	content := "# Title\n\n**bold** and `code`"
	lines := renderAssistantBlock(content, 60, nil)
	joined := strings.Join(lines, "\n")
	if !headingTextPresent(joined, "Title") || !strings.Contains(joined, "bold") {
		t.Fatalf("expected rendered markdown text:\n%s", joined)
	}
	if !strings.Contains(markdown.StripANSI(lines[0]), AssistantBullet) {
		t.Fatalf("first line should keep assistant bullet:\n%s", joined)
	}
}

func TestRenderUserFullWidthBackground(t *testing.T) {
	blocks := []Block{{Role: RoleUser}}
	blocks[0].Content = "hi"

	out := Render(blocks, 30, time.Now(), false, tool.DisplayContext{})
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 1 {
		t.Fatalf("got %d lines, want 1:\n%s", len(lines), out)
	}
	if lipgloss.Width(lines[0]) != 30 {
		t.Fatalf("user line width = %d, want 30:\n%q", lipgloss.Width(lines[0]), lines[0])
	}
}

func headingTextPresent(out, text string) bool {
	plain := strings.ReplaceAll(markdown.StripANSI(out), " ", "")
	compact := strings.ReplaceAll(text, " ", "")
	return strings.Contains(plain, compact)
}
