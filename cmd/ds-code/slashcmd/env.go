package slashcmd

import (
	"context"
	"io"

	"github.com/wzhejunqiu/ds-code/internal/agent"
	"github.com/wzhejunqiu/ds-code/internal/config"
	ctxpkg "github.com/wzhejunqiu/ds-code/internal/context"
	"github.com/wzhejunqiu/ds-code/internal/session"
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
