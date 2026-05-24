package session_test

import (
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/role"
	"github.com/wzhejunqiu/ds-code/internal/session"
)

func TestSplitUserTurns(t *testing.T) {
	msgs := []session.Message{
		{ID: 1, Role: role.User, Content: "hi"},
		{ID: 2, Role: role.Assistant, Content: "hello"},
		{ID: 3, Role: role.Tool, ToolName: "read_file", Content: "ok"},
		{ID: 4, Role: role.User, Content: "next"},
	}
	turns := session.SplitUserTurns(msgs)
	if len(turns) != 2 {
		t.Fatalf("turns = %d, want 2", len(turns))
	}
	if turns[0].MaxMessageID() != 3 {
		t.Fatalf("turn0 max id = %d", turns[0].MaxMessageID())
	}
	if turns[1].FirstUserContent() != "next" {
		t.Fatalf("turn1 user = %q", turns[1].FirstUserContent())
	}
}
