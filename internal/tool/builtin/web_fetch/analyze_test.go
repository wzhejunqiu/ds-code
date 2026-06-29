package web_fetch_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/datadir"
	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/llm/mock"
	"github.com/wzhejunqiu/ds-code/internal/testutil"
	"github.com/wzhejunqiu/ds-code/internal/tool/builtin/web_fetch"
	"github.com/wzhejunqiu/ds-code/internal/version"
)

func TestAnalyzePage_sendsPromptAndContent(t *testing.T) {
	llmMock := &mock.Client{Responses: []*llm.Response{{Content: "answer", FinishReason: "stop"}}}
	cfg := &config.Config{
		Web: config.WebConfig{FetchModel: "deepseek-v4-flash"},
		LLM: config.LLMConfig{MaxTokens: 4096},
	}
	out, err := web_fetch.AnalyzePage(context.Background(), llmMock, cfg, "extract title", "# Hello")
	if err != nil {
		t.Fatal(err)
	}
	if out != "answer" {
		t.Fatalf("out = %q", out)
	}
	if len(llmMock.Calls) != 1 {
		t.Fatal("expected one LLM call")
	}
	if !strings.Contains(llmMock.Calls[0].Messages[0].Content, "extract title") {
		t.Fatal("prompt missing from user message")
	}
	if !strings.Contains(llmMock.Calls[0].Messages[0].Content, "# Hello") {
		t.Fatal("markdown missing from user message")
	}
	if llmMock.Calls[0].Model != "deepseek-v4-flash" {
		t.Fatalf("model = %q", llmMock.Calls[0].Model)
	}
}

func TestAnalyzePage_usesInstallIdentifier(t *testing.T) {
	home := testutil.IsolatedHome(t)
	datadir.ResetIdentifierForTest()
	want := strings.Repeat("ab", 32)
	dir := filepath.Join(home, version.UserDataDirName)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "identifier"), []byte(want), 0o600); err != nil {
		t.Fatal(err)
	}

	llmMock := &mock.Client{Responses: []*llm.Response{{Content: "ok", FinishReason: "stop"}}}
	cfg := &config.Config{Web: config.WebConfig{FetchModel: "deepseek-v4-flash"}, LLM: config.LLMConfig{MaxTokens: 4096}}
	_, err := web_fetch.AnalyzePage(context.Background(), llmMock, cfg, "q", "body")
	if err != nil {
		t.Fatal(err)
	}
	if llmMock.Calls[0].UserID != want {
		t.Fatalf("UserID = %q, want %q", llmMock.Calls[0].UserID, want)
	}
}
