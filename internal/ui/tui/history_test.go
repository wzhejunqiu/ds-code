package tui

import (
	"testing"
	"time"

	"github.com/hejunqiu/ds-code/internal/role"
	"github.com/hejunqiu/ds-code/internal/session"
)

func TestChatBlocksFromMessages(t *testing.T) {
	msgs := []session.Message{
		{Role: role.User, Content: "hello"},
		{Role: role.Assistant, Content: "hi", ReasoningContent: "think"},
		{Role: role.Tool, Content: "<tool_result name=\"read_file\" id=\"1\">\noutput\n</tool_result>", ToolName: "read_file", ToolCallID: "1"},
		{Role: role.System, Content: "rewound"},
		{Role: role.Assistant, Content: "", ToolCallsJSON: `[{"id":"1","name":"read_file","arguments":"{\"path\":\"a.go\"}"}]`},
		{Role: role.Tool, Content: "<tool_result name=\"read_file\" id=\"1\">\nmore\n</tool_result>", ToolName: "read_file", ToolCallID: "1"},
	}
	blocks := chatBlocksFromMessages(msgs, true)
	if len(blocks) != 3 {
		t.Fatalf("got %d blocks, want 3", len(blocks))
	}
	if blocks[0].role != chatRoleUser || blocks[0].content.String() != "hello" {
		t.Fatalf("user block: %+v", blocks[0])
	}
	if blocks[1].role != chatRoleAssistant || blocks[1].content.String() != "hi" || blocks[1].reasoning.String() != "think" {
		t.Fatalf("assistant block: %+v", blocks[1])
	}
	if !blocks[1].reasoningOpen {
		t.Fatal("expected reasoning open")
	}
	if blocks[2].role != chatRoleTool || blocks[2].toolName != "read_file" || blocks[2].toolResult != "more" {
		t.Fatalf("tool block: %+v", blocks[2])
	}
	if blocks[2].toolArgs != "path=a.go" {
		t.Fatalf("tool args = %q", blocks[2].toolArgs)
	}
}

func TestChatBlocksFromMessages_reasoningBeforeTools(t *testing.T) {
	msgs := []session.Message{
		{Role: role.Assistant, ReasoningContent: "think first", ToolCallsJSON: `[{"id":"1","name":"read_file","arguments":"{\"path\":\"a.go\"}"}]`},
		{Role: role.Tool, Content: "body", ToolName: "read_file", ToolCallID: "1"},
	}
	blocks := chatBlocksFromMessages(msgs, true)
	if len(blocks) != 2 {
		t.Fatalf("got %d blocks, want 2", len(blocks))
	}
	if blocks[0].role != chatRoleAssistant || blocks[0].reasoning.String() != "think first" {
		t.Fatalf("assistant block first: %+v", blocks[0])
	}
	if blocks[1].role != chatRoleTool {
		t.Fatalf("tool block second: %+v", blocks[1])
	}
}

func TestChatBlocksFromMessages_interruptSystemMessage(t *testing.T) {
	msgs := []session.Message{
		{Role: role.User, Content: "hello"},
		{Role: role.System, Content: interruptSessionMarker()},
	}
	blocks := chatBlocksFromMessages(msgs, true)
	if len(blocks) != 2 {
		t.Fatalf("got %d blocks, want 2", len(blocks))
	}
	if blocks[1].role != chatRoleInterrupt {
		t.Fatalf("second block role = %s, want interrupt", blocks[1].role)
	}
}

func TestChatBlocksFromMessages_durations(t *testing.T) {
	msgs := []session.Message{
		{Role: role.Assistant, Content: "hi", ReasoningContent: "think", ReasoningDurationMS: 1200, TurnDurationMS: 5000},
	}
	blocks := chatBlocksFromMessages(msgs, false)
	if len(blocks) != 1 {
		t.Fatalf("got %d blocks", len(blocks))
	}
	if blocks[0].reasoningDuration != 1200*time.Millisecond {
		t.Fatalf("reasoningDuration = %v", blocks[0].reasoningDuration)
	}
	if blocks[0].turnDuration != 5*time.Second {
		t.Fatalf("turnDuration = %v", blocks[0].turnDuration)
	}
}
