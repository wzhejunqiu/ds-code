package context_test

import (
	"context"
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/config"
	ctxpkg "github.com/wzhejunqiu/ds-code/internal/context"
	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/llm/mock"
	"github.com/wzhejunqiu/ds-code/internal/role"
	"github.com/wzhejunqiu/ds-code/internal/session"
	"github.com/wzhejunqiu/ds-code/internal/tool"
)

func TestCompactAPIContext_spillHintNotInSummary(t *testing.T) {
	store := session.NewMemoryStore()
	sess, err := store.NewSession("deepseek-v4-pro", "max", "enabled", "ask", "agent")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	spillPath := "/Users/me/.ds-code/projects/abc/mcp-result/sess-old/call_abc.txt"
	toolBody := strings.Repeat("x", 500) + "\n... [MCP 完整结果已保存至 " + spillPath + "；请用 read_file 读取该绝对路径（shell 无法访问）]"

	// Turn 1 (will be compacted)
	_ = store.AppendMessage(ctx, session.Message{SessionID: sess.ID, Role: role.User, Content: "old question"})
	_ = store.AppendMessage(ctx, session.Message{SessionID: sess.ID, Role: role.Assistant, Content: "calling tool"})
	_ = store.AppendMessage(ctx, session.Message{SessionID: sess.ID, Role: role.Tool, Content: toolBody, ToolCallID: "call_abc", ToolName: "mcp_tool"})
	// Turn 2 (will be compacted)
	_ = store.AppendMessage(ctx, session.Message{SessionID: sess.ID, Role: role.User, Content: "follow up old"})
	_ = store.AppendMessage(ctx, session.Message{SessionID: sess.ID, Role: role.Assistant, Content: "old answer"})
	// Turn 3 (kept)
	_ = store.AppendMessage(ctx, session.Message{SessionID: sess.ID, Role: role.User, Content: "recent question"})
	_ = store.AppendMessage(ctx, session.Message{SessionID: sess.ID, Role: role.Assistant, Content: "recent answer"})

	mockLLM := &mock.Client{
		Responses: []*llm.Response{{Content: "Prior work used MCP tools.", FinishReason: "stop"}},
	}
	svc := &ctxpkg.Service{
		Cfg: &config.Config{
			Context: config.ContextConfig{KeepRecentTurns: 1, WindowTokens: 1_048_576},
			LLM:     config.LLMConfig{MaxTokens: 4096},
		},
		Store: store,
		Tools: tool.NewRegistry(),
		LLM:   mockLLM,
	}
	if err := svc.CompactAPIContext(ctx, sess.ID); err != nil {
		t.Fatal(err)
	}
	view, err := svc.BuildAPIContext(ctx, sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range view.Messages {
		if strings.Contains(m.Content, spillPath) {
			t.Fatalf("compact API context must not retain spill hint path in message: role=%s", m.Role)
		}
	}
}
