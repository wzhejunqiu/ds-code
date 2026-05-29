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
	if len(msgs1) != 3 || len(msgs2) != 3 {
		t.Fatalf("expected 2 tool + 1 user messages, got %d and %d", len(msgs1), len(msgs2))
	}
	for i := 0; i < 2; i++ {
		if msgs1[i].Role != role.Tool || msgs2[i].Role != role.Tool {
			t.Fatalf("expected tool messages at index %d", i)
		}
		if msgs1[i].Content != spawn.ForkPlaceholder || msgs2[i].Content != spawn.ForkPlaceholder {
			t.Fatal("placeholder content must be byte-identical")
		}
	}
	u1 := msgs1[2].Content
	u2 := msgs2[2].Content
	before1 := u1[:strings.Index(u1, "[directive:")]
	before2 := u2[:strings.Index(u2, "[directive:")]
	if before1 != before2 {
		t.Fatalf("user prefix mismatch:\n%q\nvs\n%q", before1, before2)
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
