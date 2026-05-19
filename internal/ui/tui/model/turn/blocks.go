package turn

import (
	"time"

	"github.com/hejunqiu/ds-code/internal/ui/tui/chat"
	"github.com/hejunqiu/ds-code/internal/ui/tui/model/state"
)

func AppendPlanningBlock(s *state.State) {
	if len(s.Chat) > 0 && s.Chat[len(s.Chat)-1].Role == chat.RolePlanning {
		return
	}
	ClearPlanningBlock(s)
	s.Chat = append(s.Chat, chat.Block{
		Role:              chat.RolePlanning,
		Streaming:         true,
		PlanningStartedAt: time.Now(),
	})
}

func ClearPlanningBlock(s *state.State) {
	for i := len(s.Chat) - 1; i >= 0; i-- {
		if s.Chat[i].Role == chat.RolePlanning {
			s.Chat = append(s.Chat[:i], s.Chat[i+1:]...)
			return
		}
	}
}

func NeedsPlanningTick(s *state.State) bool {
	if !s.Running || len(s.Chat) == 0 {
		return false
	}
	return s.Chat[len(s.Chat)-1].Role == chat.RolePlanning
}

func AppendToolBlock(s *state.State, name, args, command, result string, running, isError bool) {
	if len(s.Chat) > 0 {
		last := &s.Chat[len(s.Chat)-1]
		if last.Role == chat.RoleAssistant {
			last.FinalizeReasoning(time.Now())
			last.Streaming = false
		}
	}
	s.Chat = append(s.Chat, chat.Block{
		Role:        chat.RoleTool,
		ToolName:    name,
		ToolArgs:    args,
		ToolCommand: command,
		ToolResult:  result,
		ToolRunning: running,
		ToolError:   isError,
	})
}

func FinishToolBlock(s *state.State, name, args, command, result string, isError bool) {
	for i := len(s.Chat) - 1; i >= 0; i-- {
		if s.Chat[i].Role != chat.RoleTool || !s.Chat[i].ToolRunning {
			continue
		}
		s.Chat[i].ToolName = name
		s.Chat[i].ToolArgs = args
		s.Chat[i].ToolCommand = command
		s.Chat[i].ToolResult = result
		s.Chat[i].ToolRunning = false
		s.Chat[i].ToolError = isError
		return
	}
	AppendToolBlock(s, name, args, command, result, false, isError)
}

func FinalizeLastAssistant(s *state.State, at time.Time) {
	for i := len(s.Chat) - 1; i >= 0; i-- {
		if s.Chat[i].Role != chat.RoleAssistant {
			continue
		}
		s.Chat[i].FinalizeReasoning(at)
		s.Chat[i].Streaming = false
		return
	}
}
