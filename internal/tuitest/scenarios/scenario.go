//go:build tuitest

package scenarios

import (
	"time"

	"github.com/hejunqiu/ds-code/internal/llm"
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
