package agent

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	ctxpkg "github.com/wzhejunqiu/ds-code/internal/context"
	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/role"
	"github.com/wzhejunqiu/ds-code/internal/session"
	"github.com/wzhejunqiu/ds-code/internal/toolresult"
)

func TestHydrateReadPaths(t *testing.T) {
	root := t.TempDir()
	fp := filepath.Join(root, "a.go")
	if err := os.WriteFile(fp, []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	args, _ := json.Marshal(map[string]string{"filepath": fp})
	tcJSON, _ := json.Marshal([]llm.ToolCall{{
		ID: "c1", Name: "read_file", Arguments: string(args),
	}})
	msgs := []session.Message{
		{
			Role:          role.Assistant,
			ToolCallsJSON: string(tcJSON),
		},
		{
			Role:       role.Tool,
			ToolName:   "read_file",
			ToolCallID: "c1",
			Content:    ctxpkg.FormatToolResult("read_file", "c1", "1|x"),
		},
	}
	got := HydrateReadPaths(root, msgs)
	if len(got) != 1 {
		t.Fatalf("hydrated = %d paths", len(got))
	}
}

func TestHydrateReadPaths_skipsErrors(t *testing.T) {
	root := t.TempDir()
	fp := filepath.Join(root, "a.go")
	args, _ := json.Marshal(map[string]string{"filepath": fp})
	tcJSON, _ := json.Marshal([]llm.ToolCall{{
		ID: "c1", Name: "read_file", Arguments: string(args),
	}})
	msgs := []session.Message{
		{
			Role:          role.Assistant,
			ToolCallsJSON: string(tcJSON),
		},
		{
			Role:       role.Tool,
			ToolName:   "read_file",
			ToolCallID: "c1",
			Content:    toolresult.FormatToolError("read_file", "c1", os.ErrNotExist),
		},
	}
	if len(HydrateReadPaths(root, msgs)) != 0 {
		t.Fatal("expected no paths from failed read")
	}
}

func TestRunner_readPathSnapshot_hydratesOnce(t *testing.T) {
	root := t.TempDir()
	fp := filepath.Join(root, "a.go")
	if err := os.WriteFile(fp, []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := session.NewMemoryStore()
	sess, err := store.NewSession("m", "max", "enabled", "auto", "agent")
	if err != nil {
		t.Fatal(err)
	}
	args, _ := json.Marshal(map[string]string{"filepath": fp})
	tcJSON, _ := json.Marshal([]llm.ToolCall{{
		ID: "c1", Name: "read_file", Arguments: string(args),
	}})
	if err := store.AppendMessage(context.Background(), session.Message{
		SessionID:     sess.ID,
		Role:          role.Assistant,
		ToolCallsJSON: string(tcJSON),
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.AppendMessage(context.Background(), session.Message{
		SessionID:  sess.ID,
		Role:       role.Tool,
		ToolName:   "read_file",
		ToolCallID: "c1",
		Content:    ctxpkg.FormatToolResult("read_file", "c1", "1|x"),
	}); err != nil {
		t.Fatal(err)
	}

	r := &Runner{Sessions: store, Perm: permission.NewEngine("auto", root, false)}
	snap := r.readPathSnapshot(sess.ID)
	if len(snap) != 1 {
		t.Fatalf("snapshot = %d", len(snap))
	}
}

func TestCollectSameBatchReadPaths(t *testing.T) {
	root := t.TempDir()
	fp := filepath.Join(root, "a.go")
	args, _ := json.Marshal(map[string]string{"filepath": fp})
	calls := []llm.ToolCall{{ID: "c1", Name: "read_file", Arguments: string(args)}}
	got := collectSameBatchReadPaths(root, calls)
	if len(got) != 1 {
		t.Fatalf("same batch = %d", len(got))
	}
}
