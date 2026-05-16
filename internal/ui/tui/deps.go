package tui

import (
	"context"
	"io"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hejunqiu/ds-code/internal/agent"
	"github.com/hejunqiu/ds-code/internal/config"
	ctxpkg "github.com/hejunqiu/ds-code/internal/context"
	"github.com/hejunqiu/ds-code/internal/permission"
	"github.com/hejunqiu/ds-code/internal/session"
)

// SlashFunc handles a slash command line; writes output to out.
type SlashFunc func(ctx context.Context, out io.Writer, sessionID *string, line string) (handled bool, err error)

// Deps wires the TUI to the agent stack.
type Deps struct {
	Cfg         *config.Config
	Runner      *agent.Runner
	Store       session.Store
	Context     *ctxpkg.Service
	SessionID   string
	HandleSlash SlashFunc
	PromptCh    chan permission.PromptRequest
	Events      chan<- tea.Msg
}
