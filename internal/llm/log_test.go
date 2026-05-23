package llm_test

import (
	"os"
	"strings"
	"testing"

	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/llm"
	"github.com/hejunqiu/ds-code/internal/logging"
)

func TestLogRequestDebug_redactedByDefault(t *testing.T) {
	root := t.TempDir()
	cleanup, err := logging.Setup(logging.Options{ProjectRoot: root, Verbosity: 2})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	payload := `{"model":"deepseek-v4-pro","stream":true}`
	llm.LogRequestDebug(llm.Request{
		Model:     "deepseek-v4-pro",
		MaxTokens: 4096,
		Stream:    true,
	}, []byte(payload))

	b, err := os.ReadFile(config.DefaultLogPath(root))
	if err != nil {
		t.Fatal(err)
	}
	body := string(b)
	if !strings.Contains(body, "LLM request") {
		t.Fatalf("log = %q", body)
	}
	if !strings.Contains(body, "body_bytes") {
		t.Fatalf("expected body_bytes, log = %q", body)
	}
	if strings.Contains(body, `"stream":true`) {
		t.Fatalf("full body should not be logged, got %q", body)
	}
}

func TestLogRequestDebug_fullBodyWhenAllowed(t *testing.T) {
	root := t.TempDir()
	cleanup, err := logging.Setup(logging.Options{
		ProjectRoot:        root,
		Verbosity:          2,
		AllowSensitiveData: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	payload := `{"model":"deepseek-v4-pro","stream":true}`
	llm.LogRequestDebug(llm.Request{
		Model:     "deepseek-v4-pro",
		MaxTokens: 4096,
		Stream:    true,
	}, []byte(payload))

	b, err := os.ReadFile(config.DefaultLogPath(root))
	if err != nil {
		t.Fatal(err)
	}
	body := string(b)
	if !strings.Contains(body, "stream") || !strings.Contains(body, "true") {
		t.Fatalf("expected full body in log, got %q", body)
	}
	if strings.Contains(body, "body_bytes") {
		t.Fatalf("expected body field not body_bytes, got %q", body)
	}
}

func TestLogResponseDebug(t *testing.T) {
	root := t.TempDir()
	cleanup, err := logging.Setup(logging.Options{ProjectRoot: root, Verbosity: 2})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	llm.LogResponseDebug(&llm.Response{
		Content:      "hi",
		FinishReason: "stop",
		Usage:        llm.Usage{PromptTokens: 1, CompletionTokens: 2},
	})

	b, err := os.ReadFile(config.DefaultLogPath(root))
	if err != nil {
		t.Fatal(err)
	}
	body := string(b)
	if !strings.Contains(body, "LLM response") {
		t.Fatalf("log = %q", body)
	}
	if !strings.Contains(body, "stop") {
		t.Fatalf("log = %q", body)
	}
}
