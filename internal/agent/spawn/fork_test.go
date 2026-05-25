package spawn_test

import (
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/agent/spawn"
	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/role"
)

func TestBuildForkMessages_placeholderIdentical(t *testing.T) {
	calls := []llm.ToolCall{{ID: "c1", Name: "agent"}, {ID: "c2", Name: "read_file"}}
	msgs1 := spawn.BuildForkMessages(nil, calls, "task one")
	msgs2 := spawn.BuildForkMessages(nil, calls, "task two")
	if len(msgs1) != 1 || len(msgs2) != 1 {
		t.Fatalf("expected single user message")
	}
	p1 := msgs1[0].Content
	p2 := msgs2[0].Content
	idx1 := strings.Index(p1, spawn.ForkPlaceholder)
	idx2 := strings.Index(p2, spawn.ForkPlaceholder)
	if idx1 < 0 || idx2 < 0 {
		t.Fatal("missing placeholder")
	}
	// Placeholder region must be byte-identical; only directive differs.
	before1 := p1[:strings.Index(p1, "[directive:")]
	before2 := p2[:strings.Index(p2, "[directive:")]
	if before1 != before2 {
		t.Fatalf("placeholder prefix mismatch:\n%q\nvs\n%q", before1, before2)
	}
	if !strings.Contains(p1, "[directive:") || !strings.Contains(p2, "[directive:") {
		t.Fatal("expected directive wrapper")
	}
}

func TestBuildForkMessages_PreservesHistoryWithoutTriggerAssistant(t *testing.T) {
	parentMessages := []llm.Message{
		{Role: role.User, Content: "first user question"},
		{Role: role.Assistant, Content: "first answer without tools"},
		{Role: role.User, Content: "follow-up"},
	}
	result := spawn.BuildForkMessages(parentMessages, nil, "skill task")
	if len(result) < 2 {
		t.Fatalf("expected parent history + fork user message, got %d", len(result))
	}
	if result[0].Content != "first user question" {
		t.Fatalf("first message = %q, want first user question", result[0].Content)
	}
	last := result[len(result)-1]
	if last.Role != role.User {
		t.Fatalf("last role = %s, want user", last.Role)
	}
	if strings.Contains(last.Content, spawn.ForkPlaceholder) {
		t.Fatal("skill fork should not include tool_result placeholders")
	}
	if !strings.Contains(last.Content, spawn.ForkBoilerplate) {
		t.Fatal("expected fork boilerplate in last message")
	}
}

func TestIsInForkChild_detectsBoilerplate(t *testing.T) {
	msgs := []llm.Message{{Role: role.User, Content: "hello " + spawn.ForkBoilerplate}}
	if !spawn.IsInForkChild(msgs) {
		t.Fatal("expected fork child detection")
	}
}

func TestBuildChildDirective_escapesQuotesAndNewlines(t *testing.T) {
	prompt := "fix \"bug\"\nline2"
	msgs := spawn.BuildForkMessages(nil, nil, prompt)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if !strings.Contains(msgs[0].Content, `\"bug\"`) {
		t.Fatalf("expected JSON-escaped quotes in directive, got %q", msgs[0].Content)
	}
	if !strings.Contains(msgs[0].Content, "line2") {
		t.Fatal("expected prompt content preserved")
	}
}

func TestBuildForkMessages_directiveNotDoubleWrapped(t *testing.T) {
	prompt := "fix \"bug\"\nline2"
	msgs := spawn.BuildForkMessages(nil, nil, prompt)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	content := msgs[0].Content
	if strings.Count(content, "[directive:") != 1 {
		t.Fatalf("expected exactly one directive wrapper, got %d in %q", strings.Count(content, "[directive:"), content)
	}
	if strings.Contains(content, `[directive: "Here's your specific task: "[directive:`) {
		t.Fatalf("directive appears double-wrapped: %q", content)
	}
	if !strings.Contains(content, `\"bug\"`) {
		t.Fatalf("expected JSON-escaped quotes in directive, got %q", content)
	}
}
