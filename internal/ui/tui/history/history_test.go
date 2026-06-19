package history

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/agent/spawn"
	"github.com/wzhejunqiu/ds-code/internal/role"
	"github.com/wzhejunqiu/ds-code/internal/session"
	"github.com/wzhejunqiu/ds-code/internal/tool"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/chat"
)

func TestBlocksFromMessages(t *testing.T) {
	msgs := []session.Message{
		{Role: role.User, Content: "hello"},
		{Role: role.Assistant, Content: "hi", ReasoningContent: "think"},
		{Role: role.Tool, Content: "<tool_result name=\"read_file\" id=\"1\">\noutput\n</tool_result>", ToolName: "read_file", ToolCallID: "1"},
		{Role: role.System, Content: "rewound"},
		{Role: role.Assistant, Content: "", ToolCallsJSON: `[{"id":"1","name":"read_file","arguments":"{\"path\":\"a.go\"}"}]`},
		{Role: role.Tool, Content: "<tool_result name=\"read_file\" id=\"1\">\nmore\n</tool_result>", ToolName: "read_file", ToolCallID: "1"},
	}
	blocks := BlocksFromMessages(msgs, true, "", tool.DisplayContext{})
	if len(blocks) != 3 {
		t.Fatalf("got %d blocks, want 3", len(blocks))
	}
	if blocks[0].Role != chat.RoleUser || blocks[0].Content != "hello" {
		t.Fatalf("user block: %+v", blocks[0])
	}
	if blocks[1].Role != chat.RoleAssistant || blocks[1].Content != "hi" || blocks[1].Reasoning != "think" {
		t.Fatalf("assistant block: %+v", blocks[1])
	}
	if !blocks[1].ReasoningOpen {
		t.Fatal("expected reasoning open")
	}
	if blocks[2].Role != chat.RoleTool || blocks[2].ToolName != "read_file" || blocks[2].ToolResult != "more" {
		t.Fatalf("tool block: %+v", blocks[2])
	}
	if blocks[2].ToolArgs != "Read a.go" {
		t.Fatalf("tool args = %q", blocks[2].ToolArgs)
	}
}

func TestBlocksFromMessages_reasoningBeforeTools(t *testing.T) {
	msgs := []session.Message{
		{Role: role.Assistant, ReasoningContent: "think first", ToolCallsJSON: `[{"id":"1","name":"read_file","arguments":"{\"path\":\"a.go\"}"}]`},
		{Role: role.Tool, Content: "body", ToolName: "read_file", ToolCallID: "1"},
	}
	blocks := BlocksFromMessages(msgs, true, "", tool.DisplayContext{})
	if len(blocks) != 2 {
		t.Fatalf("got %d blocks, want 2", len(blocks))
	}
	if blocks[0].Role != chat.RoleAssistant || blocks[0].Reasoning != "think first" {
		t.Fatalf("assistant block first: %+v", blocks[0])
	}
	if blocks[1].Role != chat.RoleTool {
		t.Fatalf("tool block second: %+v", blocks[1])
	}
}

func TestBlocksFromMessages_maxTurnsSoftLandingShape(t *testing.T) {
	msgs := []session.Message{
		{Role: role.User, Content: "real question"},
		{Role: role.System, Content: "[ds-code] Reached max sub-rounds (3). Summarizing progress."},
		{Role: role.Assistant, Content: "summary reply"},
	}
	blocks := BlocksFromMessages(msgs, false, "", tool.DisplayContext{})
	if len(blocks) != 2 {
		t.Fatalf("got %d blocks, want user + assistant (system event hidden)", len(blocks))
	}
	if blocks[0].Role != chat.RoleUser || blocks[0].Content != "real question" {
		t.Fatalf("first block = %+v", blocks[0])
	}
	if blocks[1].Role != chat.RoleAssistant || blocks[1].Content != "summary reply" {
		t.Fatalf("second block = %+v", blocks[1])
	}
}

func TestBlocksFromMessages_interruptSystemMessage(t *testing.T) {
	msgs := []session.Message{
		{Role: role.User, Content: "hello"},
		{Role: role.System, Content: chat.InterruptSessionMarker()},
	}
	blocks := BlocksFromMessages(msgs, true, "", tool.DisplayContext{})
	if len(blocks) != 2 {
		t.Fatalf("got %d blocks, want 2", len(blocks))
	}
	if blocks[1].Role != chat.RoleInterrupt {
		t.Fatalf("second block role = %s, want interrupt", blocks[1].Role)
	}
}

func TestBlocksFromMessages_applyPatchMultiFile(t *testing.T) {
	patch := "*** Begin Patch\n*** Update File: a.go\n@@\n-x\n+y\n*** Update File: b.go\n@@\n-z\n*** End Patch\n"
	args, err := json.Marshal(map[string]string{"patch": patch})
	if err != nil {
		t.Fatal(err)
	}
	calls, err := json.Marshal([]map[string]string{
		{"id": "p1", "name": "apply_patch", "arguments": string(args)},
	})
	if err != nil {
		t.Fatal(err)
	}
	msgs := []session.Message{
		{Role: role.Assistant, ToolCallsJSON: string(calls)},
		{Role: role.Tool, Content: "ok", ToolName: "apply_patch", ToolCallID: "p1"},
	}
	blocks := BlocksFromMessages(msgs, false, "", tool.DisplayContext{})
	var tools int
	for _, b := range blocks {
		if b.Role == chat.RoleTool {
			tools++
		}
	}
	if tools != 2 {
		t.Fatalf("got %d tool blocks, want 2 (blocks=%d)", tools, len(blocks))
	}
}

func TestBlocksFromMessages_durations(t *testing.T) {
	msgs := []session.Message{
		{Role: role.Assistant, Content: "hi", ReasoningContent: "think", ReasoningDurationMS: 1200, TurnDurationMS: 5000},
	}
	blocks := BlocksFromMessages(msgs, false, "", tool.DisplayContext{})
	if len(blocks) != 1 {
		t.Fatalf("got %d blocks", len(blocks))
	}
	if blocks[0].ReasoningDuration != 1200*time.Millisecond {
		t.Fatalf("reasoningDuration = %v", blocks[0].ReasoningDuration)
	}
	if blocks[0].TurnDuration != 5*time.Second {
		t.Fatalf("turnDuration = %v", blocks[0].TurnDuration)
	}
}

func TestBlocksFromMessages_skipsTaskNotification(t *testing.T) {
	n := spawn.Notification{
		AgentID:   "sa-1",
		ToolUseID: "tc1",
		Status:    spawn.ResultCompleted,
		Summary:   `Agent "audit" completed`,
		Result:    "done",
	}
	msgs := []session.Message{
		{Role: role.User, Content: n.Format()},
		{Role: role.Assistant, Content: "reply"},
	}
	blocks := BlocksFromMessages(msgs, false, "", tool.DisplayContext{})
	if len(blocks) != 1 {
		t.Fatalf("got %d blocks, want assistant only", len(blocks))
	}
	if blocks[0].Role != chat.RoleAssistant || blocks[0].Content != "reply" {
		t.Fatalf("block = %+v", blocks[0])
	}
}

func TestBlocksFromMessages_stripsTaskNotificationPrefix(t *testing.T) {
	n := spawn.Notification{
		AgentID:   "sa-1",
		ToolUseID: "tc1",
		Status:    spawn.ResultCompleted,
		Summary:   `Agent "audit" completed`,
	}
	msgs := []session.Message{
		{Role: role.User, Content: n.Format() + "\nreal question"},
	}
	blocks := BlocksFromMessages(msgs, false, "", tool.DisplayContext{})
	if len(blocks) != 1 {
		t.Fatalf("got %d blocks", len(blocks))
	}
	if blocks[0].Content != "real question" {
		t.Fatalf("content = %q", blocks[0].Content)
	}
}

func TestBlocksFromMessages_stripsAtRefExpansion(t *testing.T) {
	original := "请严格检查 @docs/v0.1.2 的要求"
	expanded := original + "\n\n--- @docs/v0.1.2/（目录） ---\ndocs/v0.1.2/ACCEPTANCE.md\ndocs/v0.1.2/DESIGN.md\n\n如需文件内容，请使用 read_file 或 glob 按需读取。"
	msgs := []session.Message{
		{Role: role.User, Content: expanded},
		{Role: role.Assistant, Content: "reply"},
	}
	blocks := BlocksFromMessages(msgs, false, "", tool.DisplayContext{})
	if len(blocks) != 2 {
		t.Fatalf("got %d blocks, want 2", len(blocks))
	}
	if blocks[0].Role != chat.RoleUser || blocks[0].Content != original {
		t.Fatalf("user block = %+v, want content %q", blocks[0], original)
	}
}
