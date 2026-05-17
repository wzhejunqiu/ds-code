// Handlers for agent turn tea.Msg values (stream, tools, turn start/done).
package tui

import (
	"context"
	"errors"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// updateStreamContent appends assistant reply deltas from agent.OnContentDelta.
func (m *model) updateStreamContent(msg streamContentMsg) tea.Cmd {
	if m.turnEventsAllowed() {
		m.clearPlanningBlock()
		m.appendAssistantContent(msg.delta)
		m.syncChatView()
	}
	return nil
}

// updateStreamReasoning appends thinking deltas; starts thinkingTick for live duration label.
func (m *model) updateStreamReasoning(msg streamReasoningMsg) tea.Cmd {
	if !m.turnEventsAllowed() {
		return nil
	}
	m.clearPlanningBlock()
	var cmd tea.Cmd
	if m.appendAssistantReasoning(msg.delta) {
		cmd = m.nextThinkingTickCmd()
	}
	m.syncChatView()
	return cmd
}

func (m *model) updatePlanningStart() tea.Cmd {
	if !m.turnEventsAllowed() {
		return nil
	}
	m.appendPlanningBlock()
	m.syncChatView()
	return m.nextThinkingTickCmd()
}

func (m *model) updatePlanningEnd() tea.Cmd {
	m.clearPlanningBlock()
	m.syncChatView()
	return nil
}

func (m *model) updateThinkingTick() tea.Cmd {
	if m.needsThinkingTick() || m.needsPlanningTick() {
		m.syncChatView()
		return m.nextThinkingTickCmd()
	}
	return nil
}

func (m *model) updateToolStart(msg toolStartMsg) tea.Cmd {
	if !m.turnEventsAllowed() {
		return nil
	}
	m.appendToolBlock(msg.name, msg.args, msg.command, "", true, false)
	m.toolLines = append(m.toolLines, toolLine(msg.name, msg.args, msg.command, "", true, false))
	m.syncChatView()
	m.syncToolView()
	return nil
}

// updateAssistantSegmentEnd closes the assistant segment at an agent sub-round boundary.
func (m *model) updateAssistantSegmentEnd() tea.Cmd {
	if !m.turnEventsAllowed() {
		return nil
	}
	m.finalizeLastAssistant(time.Now())
	return nil
}

func (m *model) updateToolEnd(msg toolEndMsg) tea.Cmd {
	if !m.turnEventsAllowed() {
		return nil
	}
	m.finishToolBlock(msg.name, msg.args, msg.command, msg.result, msg.isError)
	m.toolLines = m.toolLines[:0]
	for _, b := range m.chat {
		if b.role == chatRoleTool {
			preview := b.toolResult
			if preview == "" && b.toolRunning {
				preview = "…"
			}
			m.toolLines = append(m.toolLines, toolLine(b.toolName, b.toolArgs, b.toolCommand, preview, b.toolRunning, b.toolError))
		}
	}
	m.syncChatView()
	m.syncToolView()
	return nil
}

// updateTurnStarted stores cancel func; honors Esc pressed before RunTurn returned cancel.
func (m *model) updateTurnStarted(msg turnStartedMsg) tea.Cmd {
	m.turnCancel = msg.cancel
	if m.turnEscPending {
		m.turnEscPending = false
		m.cancelTurn()
	}
	return nil
}

// updateTurnDone clears running state, finalizes blocks, applies metrics, persists interrupt marker.
func (m *model) updateTurnDone(msg turnDoneMsg) tea.Cmd {
	m.running = false
	m.turnCancel = nil
	m.turnEscPending = false
	m.clearPlanningBlock()
	now := time.Now()
	m.finalizeLastAssistant(now)
	for i := range m.chat {
		if m.chat[i].role == chatRoleTool && m.chat[i].toolRunning {
			m.chat[i].toolRunning = false
		}
	}
	m.applyTurnMetrics(msg.result)
	if m.currentTurnInterrupted() {
		m.errLine = ""
		m.persistTurnInterrupt()
	} else if msg.err != nil {
		if errors.Is(msg.err, context.Canceled) {
			m.errLine = "turn cancelled"
		} else {
			m.errLine = msg.err.Error()
		}
	} else {
		m.errLine = ""
	}
	m.refreshStatus()
	m.syncChatView()
	return tea.Batch(m.listenPrompt(), statusTick())
}
