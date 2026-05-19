// Package subagentstore persists subagent (task tool) runs separately from main sessions.
package subagentstore

import (
	"context"
	"time"

	"github.com/hejunqiu/ds-code/internal/llm"
	"github.com/hejunqiu/ds-code/internal/role"
)

// Status is a subagent run lifecycle state.
type Status string

const (
	StatusRunning Status = "running"
	StatusDone    Status = "done"
	StatusError   Status = "error"
)

// Run is metadata for one task subagent execution.
type Run struct {
	ID                        string
	ParentSessionID           string
	ParentToolCallID          string
	Label                     string
	Prompt                    string
	Status                    Status
	Error                     string
	Model                     string
	ReasoningEffort           string
	ThinkingType              string
	PromptTokensTotal         int64
	CompletionTokensTotal     int64
	PromptCacheHitTokensTotal int64
	CreatedAt                 time.Time
	EndedAt                   time.Time
}

// Message is one row in a subagent transcript.
type Message struct {
	ID                   int64
	RunID                string
	Role                 role.Role
	Content              string
	ReasoningContent     string
	ReasoningDurationMS  int64
	TurnDurationMS       int64
	ToolCallsJSON        string
	ToolCallID           string
	ToolName             string
	PromptTokens         int64
	CompletionTokens     int64
	PromptCacheHitTokens int64
	CreatedAt            time.Time
}

// CreateRunParams holds fields for a new subagent run.
type CreateRunParams struct {
	ParentSessionID  string
	ParentToolCallID string
	Label            string
	Prompt           string
	Model            string
	ReasoningEffort  string
	ThinkingType     string
}

// Store persists subagent runs and messages.
type Store interface {
	CreateRun(ctx context.Context, p CreateRunParams) (Run, error)
	FinishRun(ctx context.Context, runID string, status Status, errMsg string) error
	ListRuns(ctx context.Context, parentSessionID string) ([]Run, error)
	GetRun(ctx context.Context, runID string) (Run, error)
	GetRunByToolCall(ctx context.Context, parentSessionID, parentToolCallID string) (Run, error)

	AppendMessage(ctx context.Context, msg Message) error
	ListMessages(ctx context.Context, runID string) ([]Message, error)

	AddUsage(ctx context.Context, runID string, u llm.Usage) error
	SumUsage(ctx context.Context, parentSessionID string) (llm.Usage, error)
}
