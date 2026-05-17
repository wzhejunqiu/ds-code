package tui

import (
	"time"

	"github.com/hejunqiu/ds-code/internal/ui/tui/chat"
)

// appendPlanningBlock shows "Planning next moves" between agent sub-rounds.
func (m *model) appendPlanningBlock() {
	m.clearPlanningBlock()
	m.chat = append(m.chat, chat.Block{
		Role:              chat.RolePlanning,
		Streaming:         true,
		PlanningStartedAt: time.Now(),
	})
}

func (m *model) clearPlanningBlock() {
	for i := len(m.chat) - 1; i >= 0; i-- {
		if m.chat[i].Role == chat.RolePlanning {
			m.chat = append(m.chat[:i], m.chat[i+1:]...)
			return
		}
	}
}

func (m *model) needsPlanningTick() bool {
	if !m.running || len(m.chat) == 0 {
		return false
	}
	return m.chat[len(m.chat)-1].Role == chat.RolePlanning
}

// appendToolBlock starts a tool row; finalizeLastAssistant ends any streaming assistant first.
func (m *model) appendToolBlock(name, args, command, result string, running, isError bool) {
	if len(m.chat) > 0 {
		last := &m.chat[len(m.chat)-1]
		if last.Role == chat.RoleAssistant {
			last.FinalizeReasoning(time.Now())
			last.Streaming = false
		}
	}
	m.chat = append(m.chat, chat.Block{
		Role:        chat.RoleTool,
		ToolName:    name,
		ToolArgs:    args,
		ToolCommand: command,
		ToolResult:  result,
		ToolRunning: running,
		ToolError:   isError,
	})
}

func (m *model) finishToolBlock(name, args, command, result string, isError bool) {
	for i := len(m.chat) - 1; i >= 0; i-- {
		if m.chat[i].Role != chat.RoleTool || !m.chat[i].ToolRunning {
			continue
		}
		m.chat[i].ToolName = name
		m.chat[i].ToolArgs = args
		m.chat[i].ToolCommand = command
		m.chat[i].ToolResult = result
		m.chat[i].ToolRunning = false
		m.chat[i].ToolError = isError
		return
	}
	m.appendToolBlock(name, args, command, result, false, isError)
}

func (m *model) finalizeLastAssistant(at time.Time) {
	for i := len(m.chat) - 1; i >= 0; i-- {
		if m.chat[i].Role != chat.RoleAssistant {
			continue
		}
		m.chat[i].FinalizeReasoning(at)
		m.chat[i].Streaming = false
		return
	}
}
