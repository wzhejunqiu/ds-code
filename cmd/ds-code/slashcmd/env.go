package slashcmd

import (
	"context"
	"io"

	"github.com/hejunqiu/ds-code/internal/agent"
	"github.com/hejunqiu/ds-code/internal/config"
	ctxpkg "github.com/hejunqiu/ds-code/internal/context"
	"github.com/hejunqiu/ds-code/internal/session"
)

// Env carries runtime state for a single slash command invocation.
type Env struct {
	Ctx       context.Context
	Out       io.Writer
	Cfg       *config.Config
	Runner    *agent.Runner
	Store     session.Store
	CtxSvc    *ctxpkg.Service
	SessionID *string
}
