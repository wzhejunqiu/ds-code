//go:build tuitest

package app

import (
	"context"
	"io"

	"github.com/spf13/cobra"
	"github.com/wzhejunqiu/ds-code/cmd/ds-code/slashcmd"
	"github.com/wzhejunqiu/ds-code/internal/logging"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/tuitest/mockserver"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/header"
	"github.com/wzhejunqiu/ds-code/internal/version"
	"go.uber.org/zap"
)

// RunTUIHarness starts the TUI with harness slash wiring (tuitest builds only).
func (a *App) RunTUIHarness(cmd *cobra.Command, sessionID string, reg *mockserver.Registry, startupNotices []header.Notice) error {
	logging.L().Info("starting TUI harness", zap.Bool("resume_session", sessionID != ""))
	if _, err := a.openStore(); err != nil {
		return err
	}

	runner, store, ctxSvc, err := a.NewRunner(io.Discard)
	if err != nil {
		return err
	}

	ctx := context.Background()
	if sessionID == "" {
		sess, err := slashcmd.CreateSession(a.Cfg, store)
		if err != nil {
			return err
		}
		sessionID = sess.ID
		if err := slashcmd.SeedGitSnapshot(a.Cfg, ctx, store, sessionID); err != nil {
			return err
		}
	} else if _, err := store.Get(ctx, sessionID); err != nil {
		return err
	}

	promptCh := make(chan permission.PromptRequest, 1)
	if a.Cfg.Permission.Mode == "ask" {
		runner.Perm.Interactive = true
		runner.Perm.Prompter = permission.TUIPrompter(promptCh)
	}

	subStore := a.subStore
	deps := tui.Deps{
		Cfg:            a.Cfg,
		Runner:         runner,
		Store:          store,
		Subagent:       subStore,
		Context:        ctxSvc,
		SessionID:      sessionID,
		Version:        version.Version,
		PromptCh:       promptCh,
		StartupNotices: append([]header.Notice(nil), startupNotices...),
		HandleSlash: func(c context.Context, w io.Writer, sid *string, line, activeAgentType string) (bool, error) {
			env := &slashcmd.Env{
				Ctx: c, Out: w, Cfg: a.Cfg, Runner: runner, Store: store,
				CtxSvc: ctxSvc, SessionID: sid, Spawn: a.spawnRunner(runner),
				ActiveAgentType: activeAgentType,
			}
			return slashcmd.Handle(env, a, line)
		},
	}

	_ = reg
	defer a.Close()
	return tui.Run(deps)
}
