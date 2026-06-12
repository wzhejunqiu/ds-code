package deps

import (
	"context"
	"io"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wzhejunqiu/ds-code/internal/agent"
	"github.com/wzhejunqiu/ds-code/internal/config"
	ctxpkg "github.com/wzhejunqiu/ds-code/internal/context"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/session"
	"github.com/wzhejunqiu/ds-code/internal/session/subagentstore"
)

// SlashFunc handles a slash command line; writes output to out.
type SlashFunc func(ctx context.Context, out io.Writer, sessionID *string, line, activeAgentType string) (handled bool, err error)

// Deps wires the TUI to the agent stack.
type Deps struct {
	Cfg              *config.Config
	Runner           *agent.Runner
	Store            session.Store
	Subagent         subagentstore.Store
	Context          *ctxpkg.Service
	SessionID        string
	Version          string
	HandleSlash      SlashFunc
	OnSessionEnd     func(sessionID string)
	PromptCh         chan permission.PromptRequest
	Events           chan<- tea.Msg
	BackgroundAgents func() int
}
