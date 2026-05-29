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
		"HTTP 413",
		"status code 413",
		"status: 413",
		"error 413",
		"request too large",
		"payload too large",
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
		"error code 4130",
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

func TestIsTransientNetworkError_positive(t *testing.T) {
	cases := []string{
		"connection reset by peer",
		"connection refused",
		"read: EOF",
		"broken pipe",
		"i/o timeout",
		"network timeout",
		"dial tcp: connection closed",
	}
	for _, msg := range cases {
		if !IsTransientNetworkError(errors.New(msg)) {
			t.Fatalf("expected true for %q", msg)
		}
	}
}

func TestIsTransientNetworkError_negative(t *testing.T) {
	cases := []string{
		"context length exceeded",
		"invalid request",
		"rate limit exceeded",
		"request timeout waiting for context",
		"context deadline exceeded",
		"",
	}
	for _, msg := range cases {
		if IsTransientNetworkError(errors.New(msg)) {
			t.Fatalf("expected false for %q", msg)
		}
	}
	if IsTransientNetworkError(nil) {
		t.Fatal("nil should be false")
	}
}
