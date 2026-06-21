package turn

import (
	"time"

	"github.com/wzhejunqiu/ds-code/internal/ui/tui/chat"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/state"
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

func AppendToolBlock(s *state.State, name, args, command, result string, running, isError bool, timeoutDeadline time.Time) {
	if len(s.Chat) > 0 {
		last := &s.Chat[len(s.Chat)-1]
		if last.Role == chat.RoleAssistant {
			last.FinalizeReasoning(time.Now())
			last.Streaming = false
		}
	}
	s.Chat = append(s.Chat, chat.Block{
		Role:                chat.RoleTool,
		ToolName:            name,
		ToolArgs:            args,
		ToolCommand:         command,
		ToolResult:          result,
		ToolRunning:         running,
		ToolError:           isError,
		ToolTimeoutDeadline: timeoutDeadline,
	})
}

func FinishToolBlock(s *state.State, name, args, command, result string, isError bool) {
	// Exact match (e.g. apply_patch per-file rows).
	for i := len(s.Chat) - 1; i >= 0; i-- {
		if s.Chat[i].Role != chat.RoleTool || !s.Chat[i].ToolRunning {
			continue
		}
		if s.Chat[i].ToolName != name || s.Chat[i].ToolArgs != args || s.Chat[i].ToolCommand != command {
			continue
		}
		finishToolAt(s, i, args, command, result, isError)
		return
	}
	// Same tool call may refresh display args on end (read_file line range, grep match count).
	for i := len(s.Chat) - 1; i >= 0; i-- {
		if s.Chat[i].Role != chat.RoleTool || !s.Chat[i].ToolRunning {
			continue
		}
		if s.Chat[i].ToolName != name || s.Chat[i].ToolCommand != command {
			continue
		}
		finishToolAt(s, i, args, command, result, isError)
		return
	}
	AppendToolBlock(s, name, args, command, result, false, isError, time.Time{})
}

func finishToolAt(s *state.State, i int, args, command, result string, isError bool) {
	s.Chat[i].ToolArgs = args
	s.Chat[i].ToolCommand = command
	s.Chat[i].ToolResult = result
	s.Chat[i].ToolRunning = false
	s.Chat[i].ToolError = isError
	s.Chat[i].ToolTimeoutDeadline = time.Time{}
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
