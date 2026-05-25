package context

import (
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/role"
)

func TestMicroCompress_replacesLargeToolResults(t *testing.T) {
	big := strings.Repeat("x", microThreshold+100)
	msgs := []llm.Message{
		{Role: role.Tool, Content: big},
		{Role: role.Tool, Content: "small"},
	}
	out := MicroCompress(msgs)
	if out[0].Content == big {
		t.Fatal("expected large tool result to be digested")
	}
	if !strings.Contains(out[0].Content, "digest") {
		t.Fatalf("unexpected digest format: %q", out[0].Content)
	}
	if out[1].Content != "small" {
		t.Fatalf("small result should be unchanged, got %q", out[1].Content)
	}
}
