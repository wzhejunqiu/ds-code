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

	"github.com/hejunqiu/ds-code/internal/agent"
	"github.com/hejunqiu/ds-code/internal/audit"
	"github.com/hejunqiu/ds-code/internal/config"
	ctxpkg "github.com/hejunqiu/ds-code/internal/context"
	"github.com/hejunqiu/ds-code/internal/llm/deepseek"
	"github.com/hejunqiu/ds-code/internal/permission"
	"github.com/hejunqiu/ds-code/internal/session"
	"github.com/hejunqiu/ds-code/internal/tool"
	"github.com/hejunqiu/ds-code/internal/tool/builtin"
	"github.com/hejunqiu/ds-code/internal/ui/slash"
	"github.com/spf13/cobra"
)

type app struct {
	cfg   *config.Config
	store session.Store
}

func (a *app) openStore() (session.Store, error) {
	if a.store != nil {
		return a.store, nil
	}
	st, err := session.OpenDefaultStore(a.cfg.ProjectRoot)
	if err != nil {
		return nil, err
	}
	a.store = st
	return st, nil
}

func (a *app) newRunner(out io.Writer) (*agent.Runner, session.Store, *ctxpkg.Service, error) {
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
	reg := tool.NewRegistry()
	reg.Register(&builtin.ReadFileTool{Cfg: a.cfg, Perm: perm, Strict: strict})
	reg.Register(&builtin.GrepTool{Cfg: a.cfg, Perm: perm, Gitignore: gi, Strict: strict})
	reg.Register(&builtin.ShellTool{Cfg: a.cfg, Perm: perm, Strict: strict})
	reg.Register(&builtin.ApplyPatchTool{Cfg: a.cfg, Perm: perm, Strict: strict})
	reg.Register(&builtin.WriteFileTool{Cfg: a.cfg, Perm: perm, Strict: strict})

	agentsMD, err := ctxpkg.LoadAgentsMD(a.cfg.ProjectRoot)
	if err != nil {
		return nil, nil, nil, err
	}

	llmClient := deepseek.NewClient(a.cfg)

	ctxSvc := &ctxpkg.Service{
		Cfg:      a.cfg,
		Store:    store,
		Tools:    reg,
		LLM:      llmClient,
		AgentsMD: agentsMD,
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

	runner := &agent.Runner{
		LLM:      llmClient,
		Tools:    reg,
		Perm:     perm,
		Sessions: store,
		Context:  ctxSvc,
		Cfg:      a.cfg,
		MaxTurns: maxTurns,
		Out:      out,
		Audit:    auditLog,
	}
	return runner, store, ctxSvc, nil
}

func (a *app) runNonInteractive(cmd *cobra.Command) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

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
	if err := a.seedGitSnapshot(ctx, store, ctxSvc, sess.ID); err != nil {
		return err
	}

	line := a.cfg.Prompt
	if handled, err := a.trySlashLine(ctx, out, runner, store, ctxSvc, &sess.ID, line); err != nil {
		return err
	} else if handled {
		return nil
	}

	result, err := runner.RunTurn(ctx, sess.ID, line)
	if err != nil {
		return err
	}

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
	for {
		fmt.Fprint(out, "> ")
		if !sc.Scan() {
			break
		}
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
		_, err := runner.RunTurn(ctx, sessionID, line)
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

func (a *app) createSession(store session.Store) (session.Session, error) {
	return store.CreateSession(
		a.cfg.LLM.Model,
		a.cfg.LLM.ReasoningEffort,
		a.cfg.LLM.Thinking.Type,
		a.cfg.Permission.Mode,
		a.cfg.RunMode,
	)
}
