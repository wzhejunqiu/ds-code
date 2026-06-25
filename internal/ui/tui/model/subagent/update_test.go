package subagent_test

import (
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/agent"
	"github.com/wzhejunqiu/ds-code/internal/tool"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/deps"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/msg"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/state"
	subagentui "github.com/wzhejunqiu/ds-code/internal/ui/tui/model/subagent"
	tuisub "github.com/wzhejunqiu/ds-code/internal/ui/tui/subagent"
)

func testState() *state.State {
	return &state.State{
		Running: false,
		Deps: &deps.Deps{
			Runner: &agent.Runner{Tools: tool.NewRegistry()},
		},
	}
}

func TestUpdateToolStart_whileMainTurnIdle(t *testing.T) {
	s := testState()
	s.Subagents.Start("sa-1", "probe", "work", "Explore", true)

	synced := false
	subagentui.UpdateToolStart(s, msg.SubagentToolStartMsg{
		SubagentID: "sa-1",
		Name:       "grep",
		Args:       `{"pattern":"foo"}`,
	}, func() { synced = true })

	rec := s.Subagents.Get("sa-1")
	if rec == nil {
		t.Fatal("missing subagent record")
	}
	if len(rec.Chat) < 3 {
		t.Fatalf("expected tool block appended, chat len=%d", len(rec.Chat))
	}
	last := rec.Chat[len(rec.Chat)-1]
	if last.ToolName != "grep" || !last.ToolRunning {
		t.Fatalf("tool block = %+v", last)
	}
	_ = synced
}

func TestUpdateToolEnd_whileMainTurnIdle(t *testing.T) {
	s := testState()
	s.Subagents.Start("sa-1", "probe", "work", "Explore", true)
	subagentui.UpdateToolStart(s, msg.SubagentToolStartMsg{SubagentID: "sa-1", Name: "grep"}, func() {})

	subagentui.UpdateToolEnd(s, msg.SubagentToolEndMsg{
		SubagentID: "sa-1",
		Name:       "grep",
		Result:     "matches",
	}, func() {})

	rec := s.Subagents.Get("sa-1")
	last := rec.Chat[len(rec.Chat)-1]
	if last.ToolRunning || last.ToolResult != "matches" {
		t.Fatalf("tool block = %+v", last)
	}
}

func TestUpdateEnd_whileMainTurnIdle(t *testing.T) {
	s := &state.State{Running: false}
	s.Subagents.Start("sa-1", "probe", "work", "Explore", true)

	subagentui.UpdateEnd(s, msg.SubagentEndMsg{ID: "sa-1", Summary: "report"}, func() {})

	rec := s.Subagents.Get("sa-1")
	if rec.Status != tuisub.StatusDone {
		t.Fatalf("status = %v, want done", rec.Status)
	}
}
