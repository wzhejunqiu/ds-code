//go:build tuitest

// Package scenarios defines built-in /tcase mock LLM scripts.
// Human-readable script catalog: docs/TUI_TCASE_SCRIPTS.md
package scenarios

import (
	"time"

	"github.com/wzhejunqiu/ds-code/internal/llm"
)

// StreamChunk is one SSE delta emission.
type StreamChunk struct {
	Content   string
	Reasoning string
	Delay     time.Duration
}

// Turn is one chat completion response (one Runner LLM call).
type Turn struct {
	Chunks       []StreamChunk
	ToolCalls    []llm.ToolCall
	FinishReason string
	HTTPStatus   int
	ErrBody      string
}

// Scenario is a named multi-turn mock LLM script.
type Scenario struct {
	Name   string
	Prompt string
	Turns  []Turn
}
