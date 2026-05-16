package agent_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hejunqiu/ds-code/internal/agent"
	"github.com/hejunqiu/ds-code/internal/checkpoint"
	ctxpkg "github.com/hejunqiu/ds-code/internal/context"
	"github.com/hejunqiu/ds-code/internal/llm"
	"github.com/hejunqiu/ds-code/internal/llm/mock"
	"github.com/hejunqiu/ds-code/internal/permission"
	"github.com/hejunqiu/ds-code/internal/session"
	"github.com/hejunqiu/ds-code/internal/tool"
	"github.com/hejunqiu/ds-code/internal/tool/builtin"
)

func TestRunner_createsCheckpointBeforeWriteFile(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "out.txt")
	store := session.NewMemoryStore()
	sess, _ := store.NewSession("m", "max", "enabled", "auto", "agent")
	cpStore, err := checkpoint.OpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	cfg := testConfig()
	cfg.ProjectRoot = dir
	perm := permission.NewEngine("auto", dir, false)
	reg := tool.NewRegistry()
	reg.Register(&builtin.WriteFileTool{Cfg: cfg, Perm: perm, Strict: false})

	mockLLM := &mock.Client{
		Responses: []*llm.Response{{
			ToolCalls: []llm.ToolCall{{
				ID: "c1", Name: "write_file",
				Arguments: `{"path":"out.txt","content":"new"}`,
			}},
			FinishReason: "tool_calls",
		}, {Content: "done", FinishReason: "stop"}},
	}

	r := &agent.Runner{
		LLM:         mockLLM,
		Tools:       reg,
		Perm:        perm,
		Sessions:    store,
		Context:     &ctxpkg.Service{Cfg: cfg, Store: store, Tools: reg},
		Cfg:         cfg,
		MaxTurns:    3,
		Checkpoints: cpStore,
	}

	_, err = r.RunTurn(context.Background(), sess.ID, "write", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatal("file should exist after write")
	}
	list, err := cpStore.List(context.Background(), sess.ID)
	if err != nil || len(list) != 1 {
		t.Fatalf("checkpoints = %v err=%v", list, err)
	}
}
