package tui

import (
	"context"

	"github.com/hejunqiu/ds-code/internal/agent"
	"github.com/hejunqiu/ds-code/internal/permission"
	"github.com/hejunqiu/ds-code/internal/session"
)

// Agent turn messages (produced by runTurnAsync, consumed in model_update_turn.go).

type streamContentMsg struct{ delta string }
type streamReasoningMsg struct{ delta string }
type toolStartMsg struct {
	name    string
	args    string
	command string
}
type toolEndMsg struct {
	name    string
	args    string
	command string
	result  string
	isError bool
}
type assistantSegmentEndMsg struct{} // agent sub-round boundary (before tools / next LLM)
type planningStartMsg struct{}       // between sub-rounds, before next LLM request
type planningEndMsg struct{}         // first stream delta of next sub-round
type turnDoneMsg struct {
	result *agent.TurnResult
	err    error
}
type statusRefreshMsg struct{}
type thinkingTickMsg struct{}
type promptRequestMsg struct{ req permission.PromptRequest }
type overlayCloseMsg struct{}
type contextOverlayMsg struct{ text string }
type helpOverlayMsg struct{ text string }
type slashOutputMsg struct{ text string }
type turnStartedMsg struct {
	cancel context.CancelFunc // passed to Runner.RunTurn context
}
type exitConfirmTimeoutMsg struct{}

type resumeFilterTickMsg struct {
	filter string
	seq    uint64
}

type resumeListMsg struct {
	filter   string
	seq      uint64
	sessions []session.Summary
	err      error
}

type sessionResumedMsg struct {
	sessionID string
	chat      []chatBlock
	err       error
}

type historyLoadedMsg struct { // loadInitialHistory → updateHistoryLoaded
	chat []chatBlock
	err  error
}
