package app

import (
	"context"
	"io"

	"github.com/wzhejunqiu/ds-code/cmd/ds-code/slashcmd"
	"github.com/wzhejunqiu/ds-code/internal/agent"
	ctxpkg "github.com/wzhejunqiu/ds-code/internal/context"
	"github.com/wzhejunqiu/ds-code/internal/session"
	uislash "github.com/wzhejunqiu/ds-code/internal/ui/slash"
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
		Spawn:     a.spawnRunner(runner),
	}
	return slashcmd.Handle(env, a, line)
}
