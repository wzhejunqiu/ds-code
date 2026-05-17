package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hejunqiu/ds-code/internal/agent"
	"github.com/hejunqiu/ds-code/internal/audit"
	"github.com/hejunqiu/ds-code/internal/checkpoint"
	"github.com/hejunqiu/ds-code/internal/config"
	ctxpkg "github.com/hejunqiu/ds-code/internal/context"
	"github.com/hejunqiu/ds-code/internal/llm/deepseek"
	"github.com/hejunqiu/ds-code/internal/logging"
	"github.com/hejunqiu/ds-code/internal/lsp"
	mcpsvc "github.com/hejunqiu/ds-code/internal/mcp"
	"github.com/hejunqiu/ds-code/internal/permission"
	"github.com/hejunqiu/ds-code/internal/session"
	"github.com/hejunqiu/ds-code/internal/shelljobs"
	"github.com/hejunqiu/ds-code/internal/tool"
	uipkg "github.com/hejunqiu/ds-code/internal/ui"
	"github.com/hejunqiu/ds-code/internal/ui/slash"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

type app struct {
	cfg          *config.Config
	store        session.Store
	mcpMgr       *mcpsvc.Manager
	lspMgr       *lsp.Manager
	checkpointSt *checkpoint.Store
	shellJobs    *shelljobs.Manager
}

func (a *app) openStore() (session.Store, error) {
	if a.store != nil {
		return a.store, nil
	}
	sqlite, err := session.OpenDefaultStore(a.cfg.ProjectRoot)
	if err != nil {
		return nil, err
	}
	a.store = session.NewLazyStore(sqlite)
	return a.store, nil
}

func (a *app) newRunner(out io.Writer) (*agent.Runner, session.Store, *ctxpkg.Service, error) {
	logging.L().Info("building agent runner",
		zap.String("project_root", a.cfg.ProjectRoot),
		zap.String("run_mode", a.cfg.RunMode),
		zap.String("permission", a.cfg.Permission.Mode),
	)
	store, err := a.openStore()
	if err != nil {
		return nil, nil, nil, err
	}
	interactive := permission.IsInteractiveTTY()
	perm := permission.NewEngine(a.cfg.Permission.Mode, a.cfg.ProjectRoot, interactive)
	if interactive && a.cfg.Permission.Mode == "ask" {
		perm.Prompter = permission.StdinPrompter(os.Stderr)
	}

	strict := a.cfg.LLM.StrictTools
	gi, _ := tool.LoadGitignore(a.cfg.ProjectRoot)

	if err := a.ensureMCP(context.Background(), perm, strict); err != nil {
		return nil, nil, nil, err
	}

	llmClient := deepseek.NewClient(a.cfg)
	bundle, err := a.buildTools(context.Background(), perm, gi, strict, llmClient, a.cfg.RunMode)
	if err != nil {
		return nil, nil, nil, err
	}

	agentsMD, err := ctxpkg.LoadAgentsMD(a.cfg.ProjectRoot)
	if err != nil {
		return nil, nil, nil, err
	}
	rules, err := ctxpkg.LoadRules(a.cfg.ProjectRoot)
	if err != nil {
		return nil, nil, nil, err
	}

	ctxSvc := &ctxpkg.Service{
		Cfg:      a.cfg,
		Store:    store,
		Tools:    bundle.reg,
		LLM:      llmClient,
		AgentsMD: agentsMD,
		Rules:    rules,
		AtExpander: &ctxpkg.AtExpander{
			Cfg:       a.cfg,
			Perm:      perm,
			Gitignore: gi,
		},
	}

	maxTurns := a.cfg.Agent.MaxTurns
	if maxTurns <= 0 {
		maxTurns = 25
	}

	var auditLog *audit.Logger
	if a.cfg.Audit.Enabled {
		auditLog = audit.NewLogger(config.DefaultAuditLogPath(a.cfg.ProjectRoot))
	}

	cpStore, err := a.openCheckpointStore()
	if err != nil {
		return nil, nil, nil, err
	}

	runner := &agent.Runner{
		LLM:      llmClient,
		Tools:    bundle.reg,
		Perm:     perm,
		Sessions: store,
		Context:  ctxSvc,
		Cfg:      a.cfg,
		MaxTurns: maxTurns,
		Out:      out,
		Audit:       auditLog,
		Checkpoints: cpStore,
	}
	return runner, store, ctxSvc, nil
}

func (a *app) openCheckpointStore() (*checkpoint.Store, error) {
	if a.checkpointSt != nil {
		return a.checkpointSt, nil
	}
	st, err := checkpoint.OpenStore(a.cfg.ProjectRoot)
	if err != nil {
		return nil, err
	}
	a.checkpointSt = st
	return st, nil
}

func (a *app) runNonInteractive(cmd *cobra.Command) error {
	logging.L().Info("non-interactive run", zap.Bool("json_output", a.cfg.JSONOutput))
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	defer a.closeMCP()
	defer a.closeLSP()
	defer a.closeShellJobs()

	out := cmd.OutOrStdout()
	runnerOut := io.Writer(out)
	if a.cfg.JSONOutput {
		runnerOut = io.Discard
	}
	runner, store, ctxSvc, err := a.newRunner(runnerOut)
	if err != nil {
		return err
	}

	sess, err := a.createSession(store)
	if err != nil {
		return err
	}
	logging.L().Info("session created", zap.String("session_id", sess.ID), zap.String("model", sess.Model))
	if err := a.seedGitSnapshot(ctx, store, ctxSvc, sess.ID); err != nil {
		return err
	}

	line := a.cfg.Prompt
	if handled, err := a.trySlashLine(ctx, out, runner, store, ctxSvc, &sess.ID, line); err != nil {
		return err
	} else if handled {
		return nil
	}

	result, err := runner.RunTurn(ctx, sess.ID, line, nil)
	if err != nil {
		logging.L().Error("turn failed", zap.String("session_id", sess.ID), zap.Error(err))
		return err
	}
	logging.L().Info("turn completed",
		zap.String("session_id", sess.ID),
		zap.Int("sub_rounds", result.SubRounds),
		zap.Int("prompt_tokens", result.Usage.PromptTokens),
		zap.Int("completion_tokens", result.Usage.CompletionTokens),
	)

	if a.cfg.JSONOutput {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]any{
			"session_id": sess.ID,
			"content":    result.FinalContent,
			"reasoning":  result.FinalReasoning,
			"usage":      result.Usage,
			"sub_rounds": result.SubRounds,
		})
	}
	if result.FinalContent == "" && result.FinalReasoning != "" {
		fmt.Fprintln(out, result.FinalReasoning)
	}
	return nil
}

func (a *app) runREPLWithSession(cmd *cobra.Command, sessionID string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	defer a.closeMCP()
	defer a.closeLSP()
	defer a.closeShellJobs()

	out := cmd.OutOrStdout()
	store, err := a.openStore()
	if err != nil {
		return err
	}
	if _, err := store.Get(ctx, sessionID); err != nil {
		return fmt.Errorf("session %q: %w", sessionID, err)
	}

	runner, _, ctxSvc, err := a.newRunner(out)
	if err != nil {
		return err
	}
	return a.replLoop(ctx, out, runner, store, ctxSvc, sessionID)
}

func (a *app) runREPL(cmd *cobra.Command) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	defer a.closeMCP()
	defer a.closeLSP()
	defer a.closeShellJobs()

	out := cmd.OutOrStdout()
	runner, store, ctxSvc, err := a.newRunner(out)
	if err != nil {
		return err
	}

	sess, err := a.createSession(store)
	if err != nil {
		return err
	}
	if err := a.seedGitSnapshot(ctx, store, ctxSvc, sess.ID); err != nil {
		return err
	}

	return a.replLoop(ctx, out, runner, store, ctxSvc, sess.ID)
}

func (a *app) replLoop(ctx context.Context, out io.Writer, runner *agent.Runner, store session.Store, ctxSvc *ctxpkg.Service, sessionID string) error {
	fmt.Fprintf(out, "ds-code REPL (session %s). /help for commands. /exit to quit.\n", sessionID)

	sc := bufio.NewScanner(os.Stdin)
	exitConfirmPending := false
	exitConfirmArmedAt := time.Time{}
	for {
		if exitConfirmPending && time.Since(exitConfirmArmedAt) >= uipkg.ExitConfirmTimeout {
			exitConfirmPending = false
		}
		fmt.Fprint(out, "> ")
		if !sc.Scan() {
			if sc.Err() == nil && !exitConfirmPending {
				fmt.Fprintln(out, "Press Ctrl+D again to exit.")
				exitConfirmPending = true
				exitConfirmArmedAt = time.Now()
				continue
			}
			break
		}
		exitConfirmPending = false
		line := sc.Text()
		if line == "/exit" || line == "/quit" {
			break
		}
		if line == "" {
			continue
		}

		if handled, err := a.trySlashLine(ctx, out, runner, store, ctxSvc, &sessionID, line); err != nil {
			fmt.Fprintf(out, "error: %v\n\n", err)
			continue
		} else if handled {
			fmt.Fprintln(out)
			continue
		}

		fmt.Fprintln(out)
		_, err := runner.RunTurn(ctx, sessionID, line, nil)
		if err != nil {
			fmt.Fprintf(out, "error: %v\n\n", err)
			continue
		}
		fmt.Fprintln(out)
	}
	return sc.Err()
}

func (a *app) trySlashLine(ctx context.Context, out io.Writer, runner *agent.Runner, store session.Store, ctxSvc *ctxpkg.Service, sessionID *string, line string) (bool, error) {
	if _, _, ok := slash.Parse(line); !ok {
		return false, nil
	}
	env := &slashEnv{
		ctx:       ctx,
		out:       out,
		cfg:       a,
		runner:    runner,
		store:     store,
		ctxSvc:    ctxSvc,
		sessionID: sessionID,
	}
	return a.handleSlash(env, line)
}

func (a *app) closeMCP() {
	if a.mcpMgr != nil {
		_ = a.mcpMgr.Close()
		a.mcpMgr = nil
	}
}

func (a *app) createSession(store session.Store) (session.Session, error) {
	return store.CreateSession(
		a.cfg.LLM.Model,
		a.cfg.LLM.ReasoningEffort,
		a.cfg.LLM.Thinking.Type,
		a.cfg.Permission.Mode,
		a.cfg.RunMode,
	)
}
