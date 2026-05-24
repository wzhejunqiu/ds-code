package client

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/llm"
)

func TestIsContextTooLong(t *testing.T) {
	if !IsContextTooLong(errString("context length exceeded")) {
		t.Fatal("expected true")
	}
	if IsContextTooLong(errString("network timeout")) {
		t.Fatal("expected false")
	}
	if IsContextTooLong(nil) {
		t.Fatal("nil should be false")
	}
}

type errString string

func (e errString) Error() string { return string(e) }

func TestChat_nonStream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("auth = %q", got)
		}
		_, _ = w.Write([]byte(`{
			"choices":[{"message":{"role":"assistant","content":"hi"},"finish_reason":"stop"}],
			"usage":{"prompt_tokens":1,"completion_tokens":2}
		}`))
	}))
	defer srv.Close()

	c := &Client{apiKey: "test-key", baseURL: srv.URL, http: srv.Client()}
	resp, err := c.Chat(context.Background(), llm.Request{
		Model:     "deepseek-v4-pro",
		MaxTokens: 100,
		Stream:    false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "hi" {
		t.Fatalf("content = %q", resp.Content)
	}
	if resp.Usage.PromptTokens != 1 || resp.Usage.CompletionTokens != 2 {
		t.Fatalf("usage = %+v", resp.Usage)
	}
}

func TestChat_streamAggregates(t *testing.T) {
	body := strings.Join([]string{
		`data: {"choices":[{"delta":{"content":"hel"},"finish_reason":""}]}`,
		`data: {"choices":[{"delta":{"content":"lo"},"finish_reason":""}]}`,
		`data: {"choices":[{"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":3,"completion_tokens":4}}`,
		"data: [DONE]",
	}, "\n") + "\n"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, body)
	}))
	defer srv.Close()

	var streamed []string
	c := &Client{apiKey: "k", baseURL: srv.URL, http: srv.Client()}
	resp, err := c.Chat(context.Background(), llm.Request{
		Model:     "m",
		MaxTokens: 50,
		Stream:    true,
		OnStream: func(d llm.StreamDelta) {
			if d.Content != "" {
				streamed = append(streamed, d.Content)
			}
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "hello" {
		t.Fatalf("content = %q", resp.Content)
	}
	if strings.Join(streamed, "") != "hello" {
		t.Fatalf("streamed = %v", streamed)
	}
}

func TestChat_apiError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"bad request","type":"invalid_request"}}`))
	}))
	defer srv.Close()

	c := &Client{apiKey: "k", baseURL: srv.URL, http: srv.Client()}
	_, err := c.Chat(context.Background(), llm.Request{Model: "m", MaxTokens: 10})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "bad request") {
		t.Fatalf("err = %v", err)
	}
}

func TestNewClient_strictToolsAppendsBeta(t *testing.T) {
	cfg := &config.Config{
		APIKey: "k",
		LLM: config.LLMConfig{
			BaseURL:     "https://api.example.com/v1",
			StrictTools: true,
		},
	}
	c := NewClient(cfg)
	if !strings.HasSuffix(c.baseURL, "/beta") {
		t.Fatalf("baseURL = %q, want /beta suffix", c.baseURL)
	}
}
