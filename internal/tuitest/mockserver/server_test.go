//go:build tuitest

package mockserver

import (
	"context"
	"os"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/llm/deepseek"
)

func TestMockServer_receivesBearerAuthorization(t *testing.T) {
	reg := NewRegistry()
	if err := reg.SetActive("stream-basic"); err != nil {
		t.Fatal(err)
	}
	srv := New(reg)
	defer srv.Close()

	_ = os.Setenv("DS_CODE_DEEPSEEK_API_KEY", "sk-tui-test-mock")
	key, err := config.LoadAPIKey()
	if err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		APIKey: key,
		LLM:    config.LLMConfig{BaseURL: srv.BaseURL(), Model: "deepseek-v4-pro", MaxTokens: 100},
	}
	client := deepseek.NewClient(cfg)
	_, err = client.Chat(context.Background(), llm.Request{
		Model:     "deepseek-v4-pro",
		MaxTokens: 100,
		Stream:    true,
	})
	if err != nil {
		t.Fatal(err)
	}
	got := srv.LastAuth()
	want := "Bearer " + key
	if got != want {
		t.Fatalf("Authorization = %q, want %q", got, want)
	}
}
