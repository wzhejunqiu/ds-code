package agent

import (
	"github.com/wzhejunqiu/ds-code/internal/billing"
	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/session"
)

// enrichAssistantUsage attaches per-round usage, model, price snapshot, and CNY cost to an assistant message.
func enrichAssistantUsage(msg *session.Message, modelID string, u llm.Usage) {
	snap := billing.SnapshotForModel(modelID)
	msg.ModelID = modelID
	msg.PricingSnapshotJSON = billing.MarshalSnapshot(snap)
	msg.PromptTokens = int64(u.PromptTokens)
	msg.CompletionTokens = int64(u.CompletionTokens)
	msg.PromptCacheHitTokens = int64(u.PromptCacheHitTokens)
	msg.EstimatedCostCNY = billing.EstimateCNYFromSnapshot(snap, u)
}
