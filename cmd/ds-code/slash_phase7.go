package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hejunqiu/ds-code/internal/agent"
)

func (a *app) slashCheckpoint(env *slashEnv, args string) error {
	args = strings.TrimSpace(args)
	if args == "" || args == "list" {
		return a.slashCheckpointList(env)
	}
	if strings.HasPrefix(args, "rewind ") {
		return a.slashCheckpointRewind(env, strings.TrimSpace(strings.TrimPrefix(args, "rewind")))
	}
	if n, err := strconv.Atoi(args); err == nil {
		return a.slashCheckpointRewind(env, strconv.Itoa(n))
	}
	return fmt.Errorf("usage: /checkpoint [list|rewind N]")
}

func (a *app) slashRewind(env *slashEnv, args string) error {
	args = strings.TrimSpace(args)
	if args == "" {
		return fmt.Errorf("usage: /rewind <checkpoint-id>")
	}
	return a.slashCheckpointRewind(env, args)
}

func (a *app) slashCheckpointList(env *slashEnv) error {
	if env.runner.Checkpoints == nil {
		return fmt.Errorf("checkpoint store unavailable")
	}
	list, err := env.runner.Checkpoints.List(env.ctx, *env.sessionID)
	if err != nil {
		return err
	}
	if len(list) == 0 {
		fmt.Fprintln(env.out, "No checkpoints for this session.")
		return nil
	}
	fmt.Fprintln(env.out, "Checkpoints:")
	for _, m := range list {
		fmt.Fprintf(env.out, "  #%d  %s  %s  files=%v\n",
			m.ID, m.CreatedAt.Format("2006-01-02 15:04"), m.Tool, m.Files)
	}
	return nil
}

func (a *app) slashCheckpointRewind(env *slashEnv, idStr string) error {
	id, err := strconv.Atoi(strings.TrimSpace(idStr))
	if err != nil || id <= 0 {
		return fmt.Errorf("invalid checkpoint id %q", idStr)
	}
	if err := env.runner.RewindCheckpoint(env.ctx, *env.sessionID, id); err != nil {
		return err
	}
	fmt.Fprintf(env.out, "Rewound workspace to checkpoint #%d.\n", id)
	return nil
}

func (a *app) slashBtw(env *slashEnv, args string) error {
	args = strings.TrimSpace(args)
	if args == "" {
		return fmt.Errorf("usage: /btw <question>")
	}
	opts := agent.EphemeralOpts{
		SessionID:          *env.sessionID,
		IncludeRecentTurns: a.cfg.BTW.IncludeRecentTurns,
		MaxTokens:          a.cfg.BTW.MaxTokens,
		CountTowardSession: a.cfg.BTW.CountTowardSession,
	}
	res, err := env.runner.RunEphemeral(env.ctx, args, opts)
	if err != nil {
		return err
	}
	fmt.Fprintf(env.out, "[btw]\n%s\n", res.Content)
	if res.Reasoning != "" {
		fmt.Fprintf(env.out, "\n(reasoning)\n%s\n", res.Reasoning)
	}
	fmt.Fprintf(env.out, "\nbtw tokens: in=%d out=%d\n", res.Usage.PromptTokens, res.Usage.CompletionTokens)
	return nil
}
