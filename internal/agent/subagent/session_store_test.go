package subagent

import (
	"context"
	"errors"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/session/subagentstore"
)

type ctxRecordingStore struct {
	*subagentstore.MemoryStore
	getRunCtx context.Context
}

func (s *ctxRecordingStore) GetRun(ctx context.Context, runID string) (subagentstore.Run, error) {
	s.getRunCtx = ctx
	if err := ctx.Err(); err != nil {
		return subagentstore.Run{}, err
	}
	return s.MemoryStore.GetRun(ctx, runID)
}

func TestSessionStore_Get_propagatesContext(t *testing.T) {
	mem := subagentstore.NewMemoryStore()
	wrap := &ctxRecordingStore{MemoryStore: mem}
	run, err := wrap.CreateRun(context.Background(), subagentstore.CreateRunParams{
		ParentSessionID:  "parent",
		ParentToolCallID: "call-1",
		Prompt:           "hi",
	})
	if err != nil {
		t.Fatal(err)
	}

	store := newSessionStore(wrap, run)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = store.Get(ctx, "")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Get err = %v, want context.Canceled", err)
	}
	if wrap.getRunCtx != ctx {
		t.Fatal("GetRun did not receive caller context")
	}
}
