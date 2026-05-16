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
	cfg *config.Config
}

func (a *app) newRunner(out io.Writer) (*agent.Runner, session.Store, *ctxpkg.Service, error) {
	store := session.NewMemoryStore()
	interactive := permission.IsInteractiveTTY()
	perm := permission.NewEngine(a.cfg.Permission.Mode, a.cfg.ProjectRoot, interactive)

	gi, _ := tool.LoadGitignore(a.cfg.ProjectRoot)
	reg := tool.NewRegistry()
	reg.Register(&builtin.ReadFileTool{Cfg: a.cfg, Perm: perm})
	reg.Register(&builtin.GrepTool{Cfg: a.cfg, Perm: perm, Gitignore: gi})
	reg.Register(&builtin.ShellTool{Cfg: a.cfg, Perm: perm})

	agentsMD, err := ctxpkg.LoadAgentsMD(a.cfg.ProjectRoot)
	if err != nil {
		return nil, nil, nil, err
	}

	ctxSvc := &ctxpkg.Service{
		Cfg:      a.cfg,
		Store:    store,
		Tools:    reg,
		AgentsMD: agentsMD,
		AtExpander: &ctxpkg.AtExpander{
			Cfg:       a.cfg,
			Perm:      perm,
			Gitignore: gi,
		},
	}

	llmClient := deepseek.NewClient(a.cfg)
	maxTurns := a.cfg.Agent.MaxTurns
	if maxTurns <= 0 {
		maxTurns = 25
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

	fmt.Fprintf(out, "ds-code REPL (session %s). /help for commands. /exit to quit.\n", sess.ID)

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

		if handled, err := a.trySlashLine(ctx, out, runner, store, ctxSvc, &sess.ID, line); err != nil {
			fmt.Fprintf(out, "error: %v\n\n", err)
			continue
		} else if handled {
			fmt.Fprintln(out)
			continue
		}

		fmt.Fprintln(out)
		_, err := runner.RunTurn(ctx, sess.ID, line)
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
