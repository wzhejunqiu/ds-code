package tui

import (
	"context"

	"github.com/hejunqiu/ds-code/internal/agent"
	"github.com/hejunqiu/ds-code/internal/permission"
	"github.com/hejunqiu/ds-code/internal/session"
)

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
type assistantSegmentEndMsg struct{}
type planningStartMsg struct{}
type planningEndMsg struct{}
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
	cancel context.CancelFunc
}
type exitConfirmTimeoutMsg struct{}

type resumeListMsg struct {
	sessions []session.Summary
	err      error
}

type sessionResumedMsg struct {
	sessionID string
	chat      []chatBlock
	err       error
}

type historyLoadedMsg struct {
	chat []chatBlock
	err  error
}
