package llm

import (
	"context"

	"github.com/wzhejunqiu/ds-code/internal/role"
)

// Client calls the chat completions API.
type Client interface {
	Chat(ctx context.Context, req Request) (*Response, error)
}

// Message is an OpenAI-compatible chat message.
type Message struct {
	Role             role.Role
	Content          string
	ReasoningContent string
	ToolCalls        []ToolCall
	ToolCallID       string
	Name             string
}

// ToolCall is a function tool invocation from the assistant.
type ToolCall struct {
	ID       string
	Name     string
	Arguments string
}

// ToolDef describes a tool for the API tools array.
type ToolDef struct {
	Name        string
	Description string
	Parameters  map[string]any
}

// StreamDelta is one streaming chunk from the API.
type StreamDelta struct {
	Content   string
	Reasoning string
}

// StreamFunc receives streaming deltas; nil disables incremental callbacks.
type StreamFunc func(StreamDelta)

// Request is a chat completion request.
type Request struct {
	MergedSystem    string
	Messages        []Message
	Model           string
	Tools           []ToolDef
	MaxTokens       int
	Stream          bool
	ThinkingType    string
	ReasoningEffort string
	UserID          string
	StrictTools     bool
	OnStream        StreamFunc
}

// Response is a completed chat completion (stream aggregated).
type Response struct {
	Content          string
	ReasoningContent string
	ToolCalls        []ToolCall
	FinishReason     string
	Usage            Usage
}

// Usage holds token usage from the API.
type Usage struct {
	PromptTokens         int
	CompletionTokens     int
	PromptCacheHitTokens int
}
