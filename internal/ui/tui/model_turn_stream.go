package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hejunqiu/ds-code/internal/agent"
	"github.com/hejunqiu/ds-code/internal/ui/tui/chat"
)

func (m *model) appendAssistantContent(s string) {
	blk := m.ensureStreamingAssistant()
	blk.FinalizeReasoning(time.Now())
	blk.Streaming = true
	blk.AppendContent(s)
}

// applyTurnMetrics attaches final turn stats to the assistant block that holds the
// visible reply. After tool rounds there may be a trailing empty assistant block;
// prefer the last one that actually has content.
func (m *model) applyTurnMetrics(result *agent.TurnResult) {
	if result == nil {
		return
	}
	idx := -1
	for i := len(m.chat) - 1; i >= 0; i-- {
		if m.chat[i].Role != chat.RoleAssistant {
			continue
		}
		if idx < 0 {
			idx = i
		}
		if m.chat[i].Content.Len() > 0 {
			idx = i
			break
		}
	}
	if idx < 0 {
		return
	}
	if result.FinalReasoningDuration > 0 && m.chat[idx].ReasoningDuration == 0 {
		m.chat[idx].ReasoningDuration = result.FinalReasoningDuration
	}
	if result.TurnDuration > 0 {
		m.chat[idx].TurnDuration = result.TurnDuration
	}
}

// ensureStreamingAssistant returns the block that receives streamContentMsg / streamReasoningMsg.
// A new assistant block is created after user, tool, or planning rows — not when merely toggling streaming.
func (m *model) ensureStreamingAssistant() *chat.Block {
	needNew := len(m.chat) == 0
	if !needNew {
		switch m.chat[len(m.chat)-1].Role {
		case chat.RoleTool, chat.RoleUser, chat.RolePlanning:
			needNew = true
		case chat.RoleAssistant:
			// Keep streaming into the current assistant until a tool/user/planning
			// block breaks the segment.
		default:
			needNew = true
		}
	}
	if needNew && len(m.chat) > 0 {
		if prev := &m.chat[len(m.chat)-1]; prev.Role == chat.RoleAssistant {
			prev.FinalizeReasoning(time.Now())
		}
	}
	if needNew {
		m.chat = append(m.chat, chat.Block{Role: chat.RoleAssistant, Streaming: true, ReasoningOpen: m.reasoningAll})
	}
	return &m.chat[len(m.chat)-1]
}

func (m *model) appendAssistantReasoning(s string) bool {
	blk := m.ensureStreamingAssistant()
	started := false
	if blk.ReasoningStartedAt.IsZero() {
		blk.ReasoningStartedAt = time.Now()
		started = true
	}
	blk.AppendReasoning(s)
	return started
}

func (m *model) needsThinkingTick() bool {
	if !m.running || len(m.chat) == 0 {
		return false
	}
	blk := m.chat[len(m.chat)-1]
	return blk.Role == chat.RoleAssistant && !blk.ReasoningStartedAt.IsZero() && blk.ReasoningEndedAt.IsZero()
}

func (m *model) thinkingElapsed() time.Duration {
	if len(m.chat) == 0 {
		return 0
	}
	blk := m.chat[len(m.chat)-1]
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

// nextThinkingTickCmd refreshes animated "thinking" / planning labels (fine then coarse interval).
func (m *model) nextThinkingTickCmd() tea.Cmd {
	if m.thinkingElapsed() < chat.ThinkingFineDuration {
		return thinkingTickAfter(thinkingFineTick)
	}
	return thinkingTickAfter(thinkingCoarseTick)
}
