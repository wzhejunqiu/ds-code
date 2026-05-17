// Package msg defines tea messages for the TUI model (agent turn, session, overlays).
package msg

import (
	"context"

	"github.com/hejunqiu/ds-code/internal/agent"
	"github.com/hejunqiu/ds-code/internal/permission"
	"github.com/hejunqiu/ds-code/internal/session"
	"github.com/hejunqiu/ds-code/internal/ui/tui/chat"
)

type StreamContentMsg struct{ Delta string }
type StreamReasoningMsg struct{ Delta string }
type ToolStartMsg struct {
	Name, Args, Command string
}
type ToolEndMsg struct {
	Name, Args, Command, Result string
	IsError                     bool
}
type AssistantSegmentEndMsg struct{}
type PlanningStartMsg struct{}
type PlanningEndMsg struct{}
type TurnDoneMsg struct {
	Result *agent.TurnResult
	Err    error
}
type StatusRefreshMsg struct{}
type ThinkingTickMsg struct{}
type PromptRequestMsg struct{ Req permission.PromptRequest }
type OverlayCloseMsg struct{}
type ContextOverlayMsg struct{ Text string }
type HelpOverlayMsg struct{ Text string }
type SlashOutputMsg struct{ Text string }
type TurnStartedMsg struct {
	Cancel context.CancelFunc
}
type ExitConfirmTimeoutMsg struct{}

type ResumeFilterTickMsg struct {
	Filter string
	Seq    uint64
}

type ResumeListMsg struct {
	Filter   string
	Seq      uint64
	Sessions []session.Summary
	Err      error
}

type SessionResumedMsg struct {
	SessionID string
	Chat      []chat.Block
	Err       error
}

type HistoryLoadedMsg struct {
	Chat []chat.Block
	Err  error
}
