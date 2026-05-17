package turn

import (
	"time"

	"github.com/hejunqiu/ds-code/internal/agent"
	"github.com/hejunqiu/ds-code/internal/ui/tui/chat"
	"github.com/hejunqiu/ds-code/internal/ui/tui/model/state"
)

func AppendAssistantContent(s *state.State, text string) {
	blk := ensureStreamingAssistant(s)
	blk.FinalizeReasoning(time.Now())
	blk.Streaming = true
	blk.AppendContent(text)
}

func ApplyTurnMetrics(s *state.State, result *agent.TurnResult) {
	if result == nil {
		return
	}
	idx := -1
	for i := len(s.Chat) - 1; i >= 0; i-- {
		if s.Chat[i].Role != chat.RoleAssistant {
			continue
		}
		if idx < 0 {
			idx = i
		}
		if s.Chat[i].Content.Len() > 0 {
			idx = i
			break
		}
	}
	if idx < 0 {
		return
	}
	if result.FinalReasoningDuration > 0 && s.Chat[idx].ReasoningDuration == 0 {
		s.Chat[idx].ReasoningDuration = result.FinalReasoningDuration
	}
	if result.TurnDuration > 0 {
		s.Chat[idx].TurnDuration = result.TurnDuration
	}
}

func ensureStreamingAssistant(s *state.State) *chat.Block {
	needNew := len(s.Chat) == 0
	if !needNew {
		switch s.Chat[len(s.Chat)-1].Role {
		case chat.RoleTool, chat.RoleUser, chat.RolePlanning:
			needNew = true
		case chat.RoleAssistant:
		default:
			needNew = true
		}
	}
	if needNew && len(s.Chat) > 0 {
		if prev := &s.Chat[len(s.Chat)-1]; prev.Role == chat.RoleAssistant {
			prev.FinalizeReasoning(time.Now())
		}
	}
	if needNew {
		s.Chat = append(s.Chat, chat.Block{Role: chat.RoleAssistant, Streaming: true, ReasoningOpen: s.ReasoningAll})
	}
	return &s.Chat[len(s.Chat)-1]
}

func AppendAssistantReasoning(s *state.State, text string) bool {
	blk := ensureStreamingAssistant(s)
	started := false
	if blk.ReasoningStartedAt.IsZero() {
		blk.ReasoningStartedAt = time.Now()
		started = true
	}
	blk.AppendReasoning(text)
	return started
}

func NeedsThinkingTick(s *state.State) bool {
	if !s.Running || len(s.Chat) == 0 {
		return false
	}
	blk := s.Chat[len(s.Chat)-1]
	return blk.Role == chat.RoleAssistant && !blk.ReasoningStartedAt.IsZero() && blk.ReasoningEndedAt.IsZero()
}

func ThinkingElapsed(s *state.State) time.Duration {
	if len(s.Chat) == 0 {
		return 0
	}
	blk := s.Chat[len(s.Chat)-1]
	if blk.ReasoningStartedAt.IsZero() {
		return 0
	}
	end := blk.ReasoningEndedAt
	if end.IsZero() {
		end = time.Now()
	}
	d := end.Sub(blk.ReasoningStartedAt)
	if d < 0 {
		return 0
	}
	return d
}
