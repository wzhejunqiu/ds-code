package llm

import (
	"errors"
	"testing"
)

func TestIsContextTooLong_positive(t *testing.T) {
	cases := []string{
		"context length exceeded",
		"maximum context length reached",
		"context window is too small",
		"context is too long for this model",
		"request exceeds the context limit",
		"too many tokens in context",
	}
	for _, msg := range cases {
		if !IsContextTooLong(errors.New(msg)) {
			t.Fatalf("expected true for %q", msg)
		}
	}
}

func TestIsContextTooLong_negative(t *testing.T) {
	cases := []string{
		"network timeout",
		"wrong context menu",
		"connection reset",
		"",
	}
	for _, msg := range cases {
		if IsContextTooLong(errors.New(msg)) {
			t.Fatalf("expected false for %q", msg)
		}
	}
	if IsContextTooLong(nil) {
		t.Fatal("nil should be false")
	}
}
