package subagent_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/agent/subagent"
	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/llm/mock"
	"github.com/wzhejunqiu/ds-code/internal/session"
	"github.com/wzhejunqiu/ds-code/internal/session/subagentstore"
)

func TestSessionTitleHook_survivesParentContextCancel(t *testing.T) {
	cfg := &config.Config{
		ProjectRoot: t.TempDir(),
		LLM:         config.LLMConfig{MaxTokens: 1024},
		Agent:       config.AgentConfig{MaxTurns: 3, SessionTitleSubagent: config.SessionTitleSubagentConfig{Enabled: true}},
		Tools:       config.ToolsConfig{Task: config.TaskToolConfig{SummaryMaxChars: 200}},
	}
	mockLLM := &mock.Client{
		Responses: []*llm.Response{
			{Content: "分析代码结构", FinishReason: "stop"},
		},
	}

	mainStore := session.NewMemoryStore()
	subStore := subagentstore.NewMemoryStore()
	sess, err := mainStore.CreateSession("deepseek-v4-pro", "max", "enabled", "auto", "agent")
	if err != nil {
		t.Fatal(err)
	}

	hook := subagent.NewSessionTitleHook(cfg, mockLLM, mainStore, subStore)
	if hook == nil {
		t.Fatal("expected hook")
	}

	parentCtx, cancel := context.WithCancel(context.Background())
	hook(parentCtx, sess.ID, "explain the repo layout")
	cancel()

	subagent.WaitForSessionTitle(sess.ID, subagent.SessionTitleWaitTimeout)

	got, err := mainStore.Get(context.Background(), sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "分析代码结构" {
		t.Fatalf("title = %q, want LLM title after parent cancel", got.Title)
	}
}

func TestWaitForSessionTitle_noInFlightReturnsImmediately(t *testing.T) {
	start := time.Now()
	subagent.WaitForSessionTitle("no-such-session", time.Second)
	if elapsed := time.Since(start); elapsed > 50*time.Millisecond {
		t.Fatalf("wait took %v, expected immediate return", elapsed)
	}
}

func TestWaitForSessionTitle_blocksUntilDone(t *testing.T) {
	cfg := &config.Config{
		ProjectRoot: t.TempDir(),
		LLM:         config.LLMConfig{MaxTokens: 1024},
		Agent:       config.AgentConfig{MaxTurns: 3, SessionTitleSubagent: config.SessionTitleSubagentConfig{Enabled: true}},
		Tools:       config.ToolsConfig{Task: config.TaskToolConfig{SummaryMaxChars: 200}},
	}
	mockLLM := &mock.Client{
		Responses: []*llm.Response{
			{Content: "等待测试标题", FinishReason: "stop"},
		},
	}

	mainStore := session.NewMemoryStore()
	subStore := subagentstore.NewMemoryStore()
	sess, err := mainStore.CreateSession("deepseek-v4-pro", "max", "enabled", "auto", "agent")
	if err != nil {
		t.Fatal(err)
	}

	hook := subagent.NewSessionTitleHook(cfg, mockLLM, mainStore, subStore)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		hook(context.Background(), sess.ID, "hello")
		subagent.WaitForSessionTitle(sess.ID, subagent.SessionTitleWaitTimeout)
		got, err := mainStore.Get(context.Background(), sess.ID)
		if err != nil {
			t.Errorf("get session: %v", err)
			return
		}
		if got.Title != "等待测试标题" {
			t.Errorf("title = %q, want 等待测试标题", got.Title)
		}
	}()
	wg.Wait()
}

func TestSessionTitleHook_deduplicatesConcurrentCalls(t *testing.T) {
	cfg := &config.Config{
		ProjectRoot: t.TempDir(),
		LLM:         config.LLMConfig{MaxTokens: 1024},
		Agent:       config.AgentConfig{MaxTurns: 3, SessionTitleSubagent: config.SessionTitleSubagentConfig{Enabled: true}},
		Tools:       config.ToolsConfig{Task: config.TaskToolConfig{SummaryMaxChars: 200}},
	}
	mockLLM := &mock.Client{
		Responses: []*llm.Response{
			{Content: "唯一标题", FinishReason: "stop"},
		},
	}

	mainStore := session.NewMemoryStore()
	subStore := subagentstore.NewMemoryStore()
	sess, err := mainStore.CreateSession("deepseek-v4-pro", "max", "enabled", "auto", "agent")
	if err != nil {
		t.Fatal(err)
	}

	hook := subagent.NewSessionTitleHook(cfg, mockLLM, mainStore, subStore)
	hook(context.Background(), sess.ID, "first")
	hook(context.Background(), sess.ID, "second")
	subagent.WaitForSessionTitle(sess.ID, subagent.SessionTitleWaitTimeout)

	if len(mockLLM.Calls) != 1 {
		t.Fatalf("LLM calls = %d, want 1 (deduped)", len(mockLLM.Calls))
	}
}
