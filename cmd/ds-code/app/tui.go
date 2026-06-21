package app

import (
	"context"
	"io"

	"github.com/spf13/cobra"
	"github.com/wzhejunqiu/ds-code/cmd/ds-code/slashcmd"
	"github.com/wzhejunqiu/ds-code/internal/logging"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	agenttool "github.com/wzhejunqiu/ds-code/internal/tool/builtin/agent"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui"
	"github.com/wzhejunqiu/ds-code/internal/version"
	"go.uber.org/zap"
)

// RunTUI starts the Bubble Tea interactive UI.
func (a *App) RunTUI(cmd *cobra.Command, sessionID string) error {
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
	_, _ = a.openShellJobs()
	var backgroundAgents func() int
	if at, ok := runner.Tools.Get("agent"); ok {
		if agt, ok := at.(*agenttool.AgentTool); ok {
			svc := agt.SpawnService()
			backgroundAgents = svc.BackgroundManager.RunningCount
		}
	}
	deps := tui.Deps{
		Cfg:              a.Cfg,
		Runner:           runner,
		Store:            store,
		Subagent:         subStore,
		Context:          ctxSvc,
		SessionID:        sessionID,
		Version:          version.Version,
		PromptCh:         promptCh,
		BackgroundAgents: backgroundAgents,
		StartupNotices:   buildStartupNotices(a),
		HandleSlash: func(c context.Context, w io.Writer, sid *string, line, activeAgentType string) (bool, error) {
			env := &slashcmd.Env{
				Ctx: c, Out: w, Cfg: a.Cfg, Runner: runner, Store: store,
				CtxSvc: ctxSvc, SessionID: sid, Spawn: a.spawnRunner(runner),
				ActiveAgentType: activeAgentType,
			}
			return slashcmd.Handle(env, a, line)
		},
		OnSessionEnd: func(sid string) {
			runner.EndSessionHooks(context.Background(), sid)
			a.cleanupSessionWorktrees(context.Background(), runner, sid)
		},
	}

	_ = out
	defer runner.EndSessionHooks(context.Background(), sessionID)
	defer a.cleanupSessionWorktrees(context.Background(), runner, sessionID)
	defer a.closeStore()
	defer a.closeMCP()
	defer a.closeLSP()
	defer a.closeShellJobs()
	return tui.Run(deps)
}
