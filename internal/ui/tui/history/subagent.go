package history

import (
	"context"

	"github.com/hejunqiu/ds-code/internal/session"
	"github.com/hejunqiu/ds-code/internal/session/subagentstore"
	"github.com/hejunqiu/ds-code/internal/ui/tui/subagent"
)

// LoadSubagentRegistry rebuilds in-memory subagent UI state from persisted runs.
func LoadSubagentRegistry(ctx context.Context, sub subagentstore.Store, parentSessionID string, reasoningOpen bool, workspace string) (subagent.Registry, error) {
	var reg subagent.Registry
	if sub == nil || parentSessionID == "" {
		return reg, nil
	}
	runs, err := sub.ListRuns(ctx, parentSessionID)
	if err != nil {
		return reg, err
	}
	for _, run := range runs {
		if run.RunKind == subagentstore.RunKindTitle {
			continue
		}
		msgs, err := sub.ListMessages(ctx, run.ID)
		if err != nil {
			return reg, err
		}
		sessMsgs := make([]session.Message, len(msgs))
		for i, m := range msgs {
			sessMsgs[i] = session.Message{
				ID:                   m.ID,
				SessionID:            m.RunID,
				Role:                 m.Role,
				Content:              m.Content,
				ReasoningContent:     m.ReasoningContent,
				ReasoningDurationMS:  m.ReasoningDurationMS,
				TurnDurationMS:       m.TurnDurationMS,
				ToolCallsJSON:        m.ToolCallsJSON,
				ToolCallID:           m.ToolCallID,
				ToolName:             m.ToolName,
				PromptTokens:         m.PromptTokens,
				CompletionTokens:     m.CompletionTokens,
				PromptCacheHitTokens: m.PromptCacheHitTokens,
				CreatedAt:            m.CreatedAt,
			}
		}
		chat := BlocksFromMessages(sessMsgs, reasoningOpen, workspace)
		rec := &subagent.Record{
			ID:                 run.ID,
			Label:              run.Label,
			Prompt:             run.Prompt,
			ParentToolCallID:   run.ParentToolCallID,
			Chat:               chat,
			StartedAt:          run.CreatedAt,
			EndedAt:            run.EndedAt,
			Err:                run.Error,
		}
		switch run.Status {
		case subagentstore.StatusRunning:
			rec.Status = subagent.StatusRunning
		case subagentstore.StatusError:
			rec.Status = subagent.StatusError
		default:
			rec.Status = subagent.StatusDone
		}
		reg.Add(rec)
	}
	return reg, nil
}
