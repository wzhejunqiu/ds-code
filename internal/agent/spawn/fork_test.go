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

func TestIsInForkChild_detectsBoilerplate(t *testing.T) {
	msgs := []llm.Message{{Role: role.User, Content: "hello " + spawn.ForkBoilerplate}}
	if !spawn.IsInForkChild(msgs) {
		t.Fatal("expected fork child detection")
	}
}
