package app

import (
	"context"
	"io"

	"github.com/hejunqiu/ds-code/cmd/ds-code/slashcmd"
	"github.com/hejunqiu/ds-code/internal/agent"
	ctxpkg "github.com/hejunqiu/ds-code/internal/context"
	"github.com/hejunqiu/ds-code/internal/session"
	uislash "github.com/hejunqiu/ds-code/internal/ui/slash"
)

// TrySlashLine dispatches a slash command line when applicable.
func (a *App) TrySlashLine(ctx context.Context, out io.Writer, runner *agent.Runner, store session.Store, ctxSvc *ctxpkg.Service, sessionID *string, line string) (bool, error) {
	if _, _, ok := uislash.Parse(line); !ok {
		return false, nil
	}
	env := &slashcmd.Env{
		Ctx:       ctx,
		Out:       out,
		Cfg:       a.Cfg,
		Runner:    runner,
		Store:     store,
		CtxSvc:    ctxSvc,
		SessionID: sessionID,
	}
	return slashcmd.Handle(env, a, line)
}
