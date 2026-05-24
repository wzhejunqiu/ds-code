// Package msg defines tea messages for the TUI model (agent turn, session, overlays).
package msg

import (
	"context"

	"github.com/wzhejunqiu/ds-code/internal/agent"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/session"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/chat"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/subagent"
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
type SubagentStartMsg struct {
	ID, Label, Prompt string
	AgentType         string
	Background        bool
}
type SubagentEndMsg struct {
	ID      string
	Summary string
	Err     error
}
type SubagentToolStartMsg struct {
	SubagentID          string
	Name, Args, Command string
}
type SubagentToolEndMsg struct {
	SubagentID                    string
	Name, Args, Command, Result string
	IsError                       bool
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

// TCasePickerItem is one row in the /tcase scenario picker (harness only).
type TCasePickerItem struct {
	Name string
	Desc string
}

// TCasePickerMsg opens an interactive scenario list (harness only).
type TCasePickerMsg struct{ Items []TCasePickerItem }
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
	Subagents subagent.Registry
	Err       error
}

type HistoryLoadedMsg struct {
	Chat      []chat.Block
	Subagents subagent.Registry
	Err       error
}
