package app

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hejunqiu/ds-code/cmd/ds-code/slashcmd"
	"github.com/hejunqiu/ds-code/internal/agent"
	ctxpkg "github.com/hejunqiu/ds-code/internal/context"
	"github.com/hejunqiu/ds-code/internal/session"
	uipkg "github.com/hejunqiu/ds-code/internal/ui"
	"github.com/spf13/cobra"
)

// RunREPLWithSession starts a stdin REPL bound to an existing session (legacy; no CLI entry).
func (a *App) RunREPLWithSession(cmd *cobra.Command, sessionID string) error {
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

// RunREPL starts a stdin REPL with a new session (legacy; no CLI entry).
func (a *App) RunREPL(cmd *cobra.Command) error {
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

	sess, err := slashcmd.CreateSession(a.Cfg, store)
	if err != nil {
		return err
	}
	if err := slashcmd.SeedGitSnapshot(a.Cfg, ctx, store, sess.ID); err != nil {
		return err
	}

	return a.replLoop(ctx, out, runner, store, ctxSvc, sess.ID)
}

func (a *App) replLoop(ctx context.Context, out io.Writer, runner *agent.Runner, store session.Store, ctxSvc *ctxpkg.Service, sessionID string) error {
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

		if handled, err := a.TrySlashLine(ctx, out, runner, store, ctxSvc, &sessionID, line); err != nil {
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
