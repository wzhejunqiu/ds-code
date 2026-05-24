package slashcmd

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/wzhejunqiu/ds-code/internal/agent"
	"github.com/wzhejunqiu/ds-code/internal/agent/spawn"
	"github.com/wzhejunqiu/ds-code/internal/session/subagentstore"
)

func Agent(env *Env, args string) error {
	prompt := strings.TrimSpace(args)
	if prompt == "" {
		return fmt.Errorf("usage: /agent <prompt>")
	}
	fmt.Fprintln(env.Out, "Spawning agent...")

	sub := env.CtxSvc.Subagent
	if sub == nil {
		sub = subagentstore.NewMemoryStore()
	}
	parentID := ""
	if env.SessionID != nil {
		parentID = *env.SessionID
	}

	svc := spawn.NewService(env.Cfg, env.Runner.Perm, env.Runner.Tools, env.Runner.LLM, sub)
	result, err := svc.Handle(env.Ctx, agent.ToolInvocation{
		SessionID:  parentID,
		ToolCallID: "agent-" + uuid.NewString(),
	}, spawn.Params{
		Description: truncateLabel(prompt, 48),
		Prompt:      prompt,
	}, env.Runner.Perm.Interactive)

	if err != nil {
		return err
	}
	fmt.Fprintln(env.Out, result)
	return nil
}

func truncateLabel(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
