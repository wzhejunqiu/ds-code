package slashcmd

import (
	"fmt"
	"strconv"
	"strings"
)

func Checkpoint(env *Env, args string) error {
	args = strings.TrimSpace(args)
	if args == "" || args == "list" {
		return checkpointList(env)
	}
	if strings.HasPrefix(args, "rewind ") {
		return checkpointRewind(env, strings.TrimSpace(strings.TrimPrefix(args, "rewind")))
	}
	if n, err := strconv.Atoi(args); err == nil {
		return checkpointRewind(env, strconv.Itoa(n))
	}
	return fmt.Errorf("usage: /checkpoint [list|rewind N]")
}

func Rewind(env *Env, args string) error {
	args = strings.TrimSpace(args)
	if args == "" {
		return fmt.Errorf("usage: /rewind <checkpoint-id>")
	}
	return checkpointRewind(env, args)
}

func checkpointList(env *Env) error {
	if env.Runner.Checkpoints == nil {
		return fmt.Errorf("checkpoint store unavailable")
	}
	list, err := env.Runner.Checkpoints.List(env.Ctx, *env.SessionID)
	if err != nil {
		return err
	}
	if len(list) == 0 {
		fmt.Fprintln(env.Out, "No checkpoints for this session.")
		return nil
	}
	fmt.Fprintln(env.Out, "Checkpoints:")
	for _, m := range list {
		fmt.Fprintf(env.Out, "  #%d  %s  %s  files=%v\n",
			m.ID, m.CreatedAt.Format("2006-01-02 15:04"), m.Tool, m.Files)
	}
	return nil
}

func checkpointRewind(env *Env, idStr string) error {
	id, err := strconv.Atoi(strings.TrimSpace(idStr))
	if err != nil || id <= 0 {
		return fmt.Errorf("invalid checkpoint id %q", idStr)
	}
	if err := env.Runner.RewindCheckpoint(env.Ctx, *env.SessionID, id); err != nil {
		return err
	}
	fmt.Fprintf(env.Out, "Rewound workspace to checkpoint #%d.\n", id)
	return nil
}
