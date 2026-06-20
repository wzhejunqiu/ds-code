package turn

import (
	"context"
	"errors"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/wzhejunqiu/ds-code/internal/logging"
	"github.com/wzhejunqiu/ds-code/internal/tool"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/chat"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/chattool"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/msg"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/state"
	"go.uber.org/zap"
)

type SyncFn func()
type NextThinkingTickFn func() tea.Cmd
type ListenPromptFn func() tea.Cmd

func UpdateStreamContent(s *state.State, m msg.StreamContentMsg, sync SyncFn) tea.Cmd {
	if !EventsAllowed(s) {
		return nil
	}
	withMainChat(s, func() {
		ClearPlanningBlock(s)
		AppendAssistantContent(s, m.Delta)
	})
	sync()
	return nil
}

func UpdateStreamReasoning(s *state.State, m msg.StreamReasoningMsg, sync SyncFn, nextTick NextThinkingTickFn) tea.Cmd {
	if !EventsAllowed(s) {
		return nil
	}
	var cmd tea.Cmd
	withMainChat(s, func() {
		ClearPlanningBlock(s)
		if AppendAssistantReasoning(s, m.Delta) {
			cmd = nextTick()
		}
	})
	sync()
	return cmd
}

func UpdatePlanningStart(s *state.State, sync SyncFn, nextTick NextThinkingTickFn) tea.Cmd {
	if !EventsAllowed(s) {
		return nil
	}
	withMainChat(s, func() { AppendPlanningBlock(s) })
	sync()
	return nextTick()
}

func UpdatePlanningEnd(s *state.State, sync SyncFn) tea.Cmd {
	if !EventsAllowed(s) {
		return nil
	}
	withMainChat(s, func() { ClearPlanningBlock(s) })
	sync()
	return nil
}

func UpdateThinkingTick(s *state.State, sync SyncFn, nextTick NextThinkingTickFn) tea.Cmd {
	if NeedsThinkingTick(s) || NeedsPlanningTick(s) {
		sync()
		return nextTick()
	}
	return nil
}

func UpdateToolStart(s *state.State, m msg.ToolStartMsg, syncChat, syncTool SyncFn) tea.Cmd {
	if !EventsAllowed(s) {
		return nil
	}
	withMainChat(s, func() {
		disp := tool.FromRegistry(s.Deps.Runner.Tools)
		AppendToolBlock(s, m.Name, m.Args, m.Command, "", true, false)
		s.ToolLines = append(s.ToolLines, chattool.Line(m.Name, m.Args, m.Command, "", true, false, disp))
	})
	syncChat()
	syncTool()
	return nil
}

func UpdateAssistantSegmentEnd(s *state.State) tea.Cmd {
	if !EventsAllowed(s) {
		return nil
	}
	FinalizeLastAssistant(s, time.Now())
	return nil
}

func UpdateToolEnd(s *state.State, m msg.ToolEndMsg, syncChat, syncTool SyncFn) tea.Cmd {
	if !EventsAllowed(s) {
		return nil
	}
	withMainChat(s, func() {
		FinishToolBlock(s, m.Name, m.Args, m.Command, m.Result, m.IsError)
		disp := tool.FromRegistry(s.Deps.Runner.Tools)
		s.ToolLines = s.ToolLines[:0]
		for i := range s.Chat {
			b := &s.Chat[i]
			if b.Role == chat.RoleTool {
				preview := b.ToolResult
				if preview == "" && b.ToolRunning {
					preview = "…"
				}
				s.ToolLines = append(s.ToolLines, chattool.Line(b.ToolName, b.ToolArgs, b.ToolCommand, preview, b.ToolRunning, b.ToolError, disp))
			}
		}
	})
	syncChat()
	syncTool()
	return nil
}

func UpdateTurnStarted(s *state.State, m msg.TurnStartedMsg, sync SyncFn) tea.Cmd {
	s.TurnCancel = m.Cancel
	if s.TurnEscPending {
		s.TurnEscPending = false
		Cancel(s, sync)
	}
	return nil
}

func UpdateTurnDone(s *state.State, m msg.TurnDoneMsg, sync SyncFn, refreshStatus func(), listen ListenPromptFn) tea.Cmd {
	s.Running = false
	s.TurnCancel = nil
	s.TurnEscPending = false
	withMainChat(s, func() {
		ClearPlanningBlock(s)
		now := time.Now()
		FinalizeLastAssistant(s, now)
		for i := range s.Chat {
			if s.Chat[i].Role == chat.RoleTool && s.Chat[i].ToolRunning {
				s.Chat[i].ToolRunning = false
			}
		}
		ApplyTurnMetrics(s, m.Result)
	})
	if CurrentTurnInterrupted(s) {
		s.ErrLine = ""
		PersistInterrupt(s)
	} else if m.Err != nil {
		if errors.Is(m.Err, context.Canceled) {
			s.ErrLine = "turn cancelled"
		} else {
			s.ErrLine = m.Err.Error()
		}
	} else {
		s.ErrLine = ""
	}
	interrupted := CurrentTurnInterrupted(s)
	promptTokens := 0
	if m.Result != nil {
		promptTokens = m.Result.Usage.PromptTokens
	}
	logging.L().Debug("tui turn done",
		zap.Bool("interrupted", interrupted),
		zap.Bool("cancelled", m.Err != nil && errors.Is(m.Err, context.Canceled)),
		zap.Bool("ok", m.Err == nil && !interrupted),
		zap.Int("prompt_tokens", promptTokens),
	)
	refreshStatus()
	sync()
	return listen()
}
