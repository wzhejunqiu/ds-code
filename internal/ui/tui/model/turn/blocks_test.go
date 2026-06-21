package turn

import (
	"testing"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/state"
)

func TestFinishToolBlock_updatesReadFileTitle(t *testing.T) {
	s := &state.State{}
	AppendToolBlock(s, "read_file", "Read sample.go", "", "", true, false, time.Time{})
	FinishToolBlock(s, "read_file", "Read sample.go L1-3", "", "1|x\n", false)
	if len(s.Chat) != 1 {
		t.Fatalf("got %d tool blocks, want 1", len(s.Chat))
	}
	if s.Chat[0].ToolArgs != "Read sample.go L1-3" {
		t.Fatalf("args = %q", s.Chat[0].ToolArgs)
	}
	if s.Chat[0].ToolRunning {
		t.Fatal("still running")
	}
}

func TestFinishToolBlock_applyPatchExactMatch(t *testing.T) {
	s := &state.State{}
	AppendToolBlock(s, "apply_patch", "a.go", "1|1", "", true, false, time.Time{})
	AppendToolBlock(s, "apply_patch", "b.go", "|2", "", true, false, time.Time{})
	FinishToolBlock(s, "apply_patch", "b.go", "|2", "ok", false)
	if s.Chat[1].ToolRunning {
		t.Fatal("b.go should be finished")
	}
	if !s.Chat[0].ToolRunning {
		t.Fatal("a.go should still be running")
	}
}
