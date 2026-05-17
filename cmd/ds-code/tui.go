package main

import (
	"context"
	"io"

	"github.com/hejunqiu/ds-code/internal/logging"
	"github.com/hejunqiu/ds-code/internal/permission"
	"github.com/hejunqiu/ds-code/internal/ui/tui"
	"github.com/hejunqiu/ds-code/internal/version"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func (a *app) runTUI(cmd *cobra.Command, sessionID string) error {
	logging.L().Info("starting TUI", zap.Bool("resume_session", sessionID != ""))
	if _, err := a.openStore(); err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	runner, store, ctxSvc, err := a.newRunner(io.Discard)
	if err != nil {
		return err
	}

	ctx := context.Background()
	if sessionID == "" {
		sess, err := a.createSession(store)
		if err != nil {
			return err
		}
		sessionID = sess.ID
		if err := a.seedGitSnapshot(ctx, store, ctxSvc, sessionID); err != nil {
			return err
		}
	} else if _, err := store.Get(ctx, sessionID); err != nil {
		return err
	}

	promptCh := make(chan permission.PromptRequest, 1)
	if a.cfg.Permission.Mode == "ask" {
		runner.Perm.Interactive = true
		runner.Perm.Prompter = permission.TUIPrompter(promptCh)
	}

	deps := tui.Deps{
		Cfg:       a.cfg,
		Runner:    runner,
		Store:     store,
		Context:   ctxSvc,
		SessionID: sessionID,
		Version:   version.Version,
		PromptCh:  promptCh,
		HandleSlash: func(c context.Context, w io.Writer, sid *string, line string) (bool, error) {
			env := &slashEnv{
				ctx: c, out: w, cfg: a, runner: runner, store: store,
				ctxSvc: ctxSvc, sessionID: sid,
			}
			return a.handleSlash(env, line)
		},
	}

	_ = out
	defer a.closeMCP()
	defer a.closeLSP()
	defer a.closeShellJobs()
	return tui.Run(deps)
}
