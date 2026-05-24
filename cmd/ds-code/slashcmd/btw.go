package slashcmd

import (
	"fmt"
	"strings"

	"github.com/wzhejunqiu/ds-code/internal/agent"
)

func Btw(env *Env, args string) error {
	args = strings.TrimSpace(args)
	if args == "" {
		return fmt.Errorf("usage: /btw <question>")
	}
	opts := agent.EphemeralOpts{
		SessionID:          *env.SessionID,
		IncludeRecentTurns: env.Cfg.BTW.IncludeRecentTurns,
		MaxTokens:          env.Cfg.BTW.MaxTokens,
		CountTowardSession: env.Cfg.BTW.CountTowardSession,
	}
	res, err := env.Runner.RunEphemeral(env.Ctx, args, opts)
	if err != nil {
		return err
	}
	fmt.Fprintf(env.Out, "[btw]\n%s\n", res.Content)
	if res.Reasoning != "" {
		fmt.Fprintf(env.Out, "\n(reasoning)\n%s\n", res.Reasoning)
	}
	fmt.Fprintf(env.Out, "\nbtw tokens: in=%d out=%d\n", res.Usage.PromptTokens, res.Usage.CompletionTokens)
	return nil
}
