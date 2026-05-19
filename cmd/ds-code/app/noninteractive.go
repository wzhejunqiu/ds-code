package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/hejunqiu/ds-code/cmd/ds-code/slashcmd"
	"github.com/hejunqiu/ds-code/internal/logging"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// RunNonInteractive runs a single prompt from -p (optionally JSON output).
func (a *App) RunNonInteractive(cmd *cobra.Command) error {
	logging.L().Info("non-interactive run", zap.Bool("json_output", a.Cfg.JSONOutput))
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	defer a.closeStore()
	defer a.closeMCP()
	defer a.closeLSP()
	defer a.closeShellJobs()

	out := cmd.OutOrStdout()
	runnerOut := io.Writer(out)
	if a.Cfg.JSONOutput {
		runnerOut = io.Discard
	}
	runner, store, ctxSvc, err := a.newRunner(runnerOut)
	if err != nil {
		return err
	}

	sess, err := slashcmd.CreateSession(a.Cfg, store)
	if err != nil {
		return err
	}
	logging.L().Info("session created", zap.String("session_id", sess.ID), zap.String("model", sess.Model))
	if err := slashcmd.SeedGitSnapshot(a.Cfg, ctx, store, sess.ID); err != nil {
		return err
	}

	line := a.Cfg.Prompt
	if handled, err := a.TrySlashLine(ctx, out, runner, store, ctxSvc, &sess.ID, line); err != nil {
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

	if a.Cfg.JSONOutput {
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
