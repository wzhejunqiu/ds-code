// Turn lifecycle: cancel/interrupt, streaming chat block assembly, tool rows, metrics.
// Bubble Tea handlers live in model_update_turn.go; agent bridge in run.go.
package tui

import (
	"context"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hejunqiu/ds-code/internal/agent"
	"github.com/hejunqiu/ds-code/internal/role"
	"github.com/hejunqiu/ds-code/internal/session"
)

// requestCancelTurn handles Esc: cancels immediately when possible, otherwise
// records intent and shows the interrupt marker until turnStartedMsg arrives.
func (m *model) requestCancelTurn() {
	if m.turnCancel != nil {
		m.turnEscPending = false
		m.cancelTurn()
		return
	}
	m.turnEscPending = true
	m.appendInterruptBlock()
}

// cancelTurn stops the in-flight agent turn (Esc). An interrupt marker is
// appended to the chat until turnDoneMsg arrives.
func (m *model) cancelTurn() {
	if m.turnCancel != nil {
		m.turnCancel()
	}
	m.dismissPrompt()
	m.appendInterruptBlock()
}

func (m *model) dismissPrompt() {
	if m.prompt == nil {
		return
	}
	select {
	case m.prompt.Reply <- false:
	default:
	}
	m.prompt = nil
	m.overlay = overlayNone
	m.overlayText = ""
}

func (m *model) appendInterruptBlock() {
	if m.currentTurnInterrupted() {
		return
	}
	now := time.Now()
	m.finalizeLastAssistant(now)
	m.clearPlanningBlock()
	for i := range m.chat {
		if m.chat[i].role == chatRoleTool && m.chat[i].toolRunning {
			m.chat[i].toolRunning = false
		}
	}
	m.chat = append(m.chat, chatBlock{role: chatRoleInterrupt})
	m.syncChatView()
}

// turnEventsAllowed gates stream/tool callbacks after Esc (interrupt marker present).
func (m *model) turnEventsAllowed() bool {
	return m.running && !m.currentTurnInterrupted()
}

// currentTurnInterrupted is true if an interrupt block exists after the latest user message.
func (m *model) currentTurnInterrupted() bool {
	lastUser := -1
	for i, b := range m.chat {
		if b.role == chatRoleUser {
			lastUser = i
		}
	}
	if lastUser < 0 {
		return false
	}
	for i := lastUser + 1; i < len(m.chat); i++ {
		if m.chat[i].role == chatRoleInterrupt {
			return true
		}
	}
	return false
}

func (m *model) persistTurnInterrupt() {
	if m.deps == nil || m.deps.Store == nil || m.sessionID == "" {
		return
	}
	_ = m.deps.Store.AppendMessage(context.Background(), session.Message{
		SessionID: m.sessionID,
		Role:      role.System,
		Content:   interruptSessionMarker(),
	})
}

func (m *model) replyPrompt(allow bool) {
	if m.prompt == nil {
		return
	}
	select {
	case m.prompt.Reply <- allow:
	default:
	}
	m.prompt = nil
	m.overlay = overlayNone
	m.overlayText = ""
}

func (m *model) handlePromptKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch strings.ToLower(msg.String()) {
	case "y", "yes":
		m.replyPrompt(true)
	case "n", "no", "esc":
		m.replyPrompt(false)
	}
	return m, m.listenPrompt()
}

// appendPlanningBlock shows "Planning next moves" between agent sub-rounds.
func (m *model) appendPlanningBlock() {
	m.clearPlanningBlock()
	m.chat = append(m.chat, chatBlock{
		role:              chatRolePlanning,
		streaming:         true,
		planningStartedAt: time.Now(),
	})
}

func (m *model) clearPlanningBlock() {
	for i := len(m.chat) - 1; i >= 0; i-- {
		if m.chat[i].role == chatRolePlanning {
			m.chat = append(m.chat[:i], m.chat[i+1:]...)
			return
		}
	}
}

func (m *model) needsPlanningTick() bool {
	if !m.running || len(m.chat) == 0 {
		return false
	}
	return m.chat[len(m.chat)-1].role == chatRolePlanning
}

// appendToolBlock starts a tool row; finalizeLastAssistant ends any streaming assistant first.
func (m *model) appendToolBlock(name, args, command, result string, running, isError bool) {
	if len(m.chat) > 0 {
		last := &m.chat[len(m.chat)-1]
		if last.role == chatRoleAssistant {
			last.finalizeReasoning(time.Now())
			last.streaming = false
		}
	}
	m.chat = append(m.chat, chatBlock{
		role:        chatRoleTool,
		toolName:    name,
		toolArgs:    args,
		toolCommand: command,
		toolResult:  result,
		toolRunning: running,
		toolError:   isError,
	})
}

func (m *model) finishToolBlock(name, args, command, result string, isError bool) {
	for i := len(m.chat) - 1; i >= 0; i-- {
		if m.chat[i].role != chatRoleTool || !m.chat[i].toolRunning {
			continue
		}
		m.chat[i].toolName = name
		m.chat[i].toolArgs = args
		m.chat[i].toolCommand = command
		m.chat[i].toolResult = result
		m.chat[i].toolRunning = false
		m.chat[i].toolError = isError
		return
	}
	m.appendToolBlock(name, args, command, result, false, isError)
}

func (m *model) finalizeLastAssistant(at time.Time) {
	for i := len(m.chat) - 1; i >= 0; i-- {
		if m.chat[i].role != chatRoleAssistant {
			continue
		}
		m.chat[i].finalizeReasoning(at)
		m.chat[i].streaming = false
		return
	}
}

func (m *model) appendAssistantContent(s string) {
	blk := m.ensureStreamingAssistant()
	blk.finalizeReasoning(time.Now())
	blk.streaming = true
	blk.appendContent(s)
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
		if m.chat[i].role != chatRoleAssistant {
			continue
		}
		if idx < 0 {
			idx = i
		}
		if m.chat[i].content.Len() > 0 {
			idx = i
			break
		}
	}
	if idx < 0 {
		return
	}
	if result.FinalReasoningDuration > 0 && m.chat[idx].reasoningDuration == 0 {
		m.chat[idx].reasoningDuration = result.FinalReasoningDuration
	}
	if result.TurnDuration > 0 {
		m.chat[idx].turnDuration = result.TurnDuration
	}
}

// ensureStreamingAssistant returns the block that receives streamContentMsg / streamReasoningMsg.
// A new assistant block is created after user, tool, or planning rows — not when merely toggling streaming.
func (m *model) ensureStreamingAssistant() *chatBlock {
	needNew := len(m.chat) == 0
	if !needNew {
		switch m.chat[len(m.chat)-1].role {
		case chatRoleTool, chatRoleUser, chatRolePlanning:
			needNew = true
		case chatRoleAssistant:
			// Keep streaming into the current assistant until a tool/user/planning
			// block breaks the segment. Do not split on content+!streaming alone — that
			// created a trailing empty assistant and hid turn duration on the wrong block.
		default:
			needNew = true
		}
	}
	if needNew && len(m.chat) > 0 {
		if prev := &m.chat[len(m.chat)-1]; prev.role == chatRoleAssistant {
			prev.finalizeReasoning(time.Now())
		}
	}
	if needNew {
		m.chat = append(m.chat, chatBlock{role: chatRoleAssistant, streaming: true, reasoningOpen: m.reasoningAll})
	}
	return &m.chat[len(m.chat)-1]
}

func (m *model) appendAssistantReasoning(s string) bool {
	blk := m.ensureStreamingAssistant()
	started := false
	if blk.reasoningStartedAt.IsZero() {
		blk.reasoningStartedAt = time.Now()
		started = true
	}
	blk.appendReasoning(s)
	return started
}

func (m *model) needsThinkingTick() bool {
	if !m.running || len(m.chat) == 0 {
		return false
	}
	blk := m.chat[len(m.chat)-1]
	return blk.role == chatRoleAssistant && !blk.reasoningStartedAt.IsZero() && blk.reasoningEndedAt.IsZero()
}

func (m *model) thinkingElapsed() time.Duration {
	if len(m.chat) == 0 {
		return 0
	}
	blk := m.chat[len(m.chat)-1]
	if blk.reasoningStartedAt.IsZero() {
		return 0
	}
	end := blk.reasoningEndedAt
	if end.IsZero() {
		end = time.Now()
	}
	d := end.Sub(blk.reasoningStartedAt)
	if d < 0 {
		return 0
	}
	return d
}

// nextThinkingTickCmd refreshes animated "thinking" / planning labels (fine then coarse interval).
func (m *model) nextThinkingTickCmd() tea.Cmd {
	if m.thinkingElapsed() < thinkingFineDuration {
		return thinkingTickAfter(thinkingFineTick)
	}
	return thinkingTickAfter(thinkingCoarseTick)
}
