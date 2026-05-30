package session

import (
	"time"

	"github.com/wzhejunqiu/ds-code/internal/role"
)

// Session holds per-conversation metadata and usage totals.
type Session struct {
	ID                        string
	Model                     string
	ReasoningEffort           string
	ThinkingType              string
	PermissionMode            PermissionMode
	RunMode                   RunMode
	CompactSummary            string
	CompactUpToMessageID      int64
	PromptTokensTotal         int64
	CompletionTokensTotal     int64
	PromptCacheHitTokensTotal int64
	PricingSnapshotJSON       string
	EstimatedCostCNY          float64
	GitSnapshot               string
	Title                     string
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
}

// BilledTokens returns prompt + completion totals for display.
func BilledTokens(s Session) int {
	return int(s.PromptTokensTotal + s.CompletionTokensTotal)
}

// Message is an append-only history row.
type Message struct {
	ID                   int64
	SessionID            string
	Role                 role.Role
	Content              string
	ReasoningContent     string
	ReasoningDurationMS  int64 // assistant: thinking phase duration
	TurnDurationMS       int64 // assistant: full user-turn wall time (final reply only)
	ToolCallsJSON        string
	ToolCallID           string
	ToolName             string
	PromptTokens         int64
	CompletionTokens     int64
	PromptCacheHitTokens int64
	ModelID              string
	PricingSnapshotJSON  string
	EstimatedCostCNY     float64
	CreatedAt            time.Time
}
