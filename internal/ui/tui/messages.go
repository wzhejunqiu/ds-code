package tui

import (
	"context"

	"github.com/hejunqiu/ds-code/internal/agent"
	"github.com/hejunqiu/ds-code/internal/permission"
)

type streamContentMsg struct{ delta string }
type streamReasoningMsg struct{ delta string }
type toolStartMsg struct{ name string }
type toolEndMsg struct{ name, preview string }
type turnDoneMsg struct {
	result *agent.TurnResult
	err    error
}
type statusRefreshMsg struct{}
type promptRequestMsg struct{ req permission.PromptRequest }
type overlayCloseMsg struct{}
type contextOverlayMsg struct{ text string }
type helpOverlayMsg struct{ text string }
type slashOutputMsg struct{ text string }
type turnStartedMsg struct {
	cancel context.CancelFunc
}
