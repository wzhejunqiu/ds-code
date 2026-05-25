package spawn_test

import (
	"context"
	"testing"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/agent/spawn"
	"github.com/wzhejunqiu/ds-code/internal/session/subagentstore"
)

func TestCleanupSessionWorktrees_removesTrackedRuns(t *testing.T) {
	store := subagentstore.NewMemoryStore()
	cfg := testConfig()
	cfg.Tools.Agent.WorktreeTTL = time.Hour
	svc := spawn.NewService(cfg, testPerm(), testRegistry(), nil, store)

	run, err := store.CreateRun(context.Background(), subagentstore.CreateRunParams{
		ParentSessionID:  "sess-a",
		ParentToolCallID: "tc1",
		AgentType:        "general-purpose",
		SpawnKind:        subagentstore.SpawnSync,
		Label:            "test",
		Prompt:           "hi",
		Model:            "deepseek-v4-pro",
		WorktreePath:     "/tmp/wt-test",
		WorktreeBranch:   "ds-code/agent-test",
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = store.FinishRun(context.Background(), run.ID, subagentstore.StatusCompleted, "")

	// Should not panic when worktree path does not exist on disk.
	svc.CleanupSessionWorktrees(context.Background(), "sess-a")
}
