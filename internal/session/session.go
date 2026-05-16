package session

import "time"

// Session holds per-conversation metadata and usage totals.
type Session struct {
	ID                        string
	Model                     string
	ReasoningEffort           string
	ThinkingType              string
	PermissionMode            string
	RunMode                   string
	CompactSummary            string
	CompactUpToMessageID      int64
	PromptTokensTotal         int64
	CompletionTokensTotal     int64
	PromptCacheHitTokensTotal int64
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
	Role                 string
	Content              string
	ReasoningContent     string
	ToolCallsJSON        string
	ToolCallID           string
	ToolName             string
	PromptTokens         int64
	CompletionTokens     int64
	PromptCacheHitTokens int64
	CreatedAt            time.Time
}
