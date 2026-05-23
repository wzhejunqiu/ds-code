package slashcmd

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/hejunqiu/ds-code/internal/agent/subagent"
	"github.com/hejunqiu/ds-code/internal/session/subagentstore"
	"github.com/hejunqiu/ds-code/internal/tool"
	"github.com/hejunqiu/ds-code/internal/tool/register"
)

func Task(env *Env, args string) error {
	prompt := strings.TrimSpace(args)
	if prompt == "" {
		return fmt.Errorf("usage: /task <prompt>")
	}
	fmt.Fprintln(env.Out, "Running read-only subagent...")

	sub := env.CtxSvc.Subagent
	if sub == nil {
		sub = subagentstore.NewMemoryStore()
	}
	parentID := ""
	if env.SessionID != nil {
		parentID = *env.SessionID
	}
	run, err := sub.CreateRun(env.Ctx, subagentstore.CreateRunParams{
		ParentSessionID:  parentID,
		ParentToolCallID: "slash-" + uuid.NewString(),
		Label:            truncateLabel(prompt, 48),
		Prompt:           prompt,
		Model:            env.Cfg.LLM.Model,
		ReasoningEffort:  env.Cfg.LLM.ReasoningEffort,
		ThinkingType:     env.Cfg.LLM.Thinking.Type,
	})
	if err != nil {
		return err
	}

	gi, _ := tool.LoadGitignore(env.Cfg.ProjectRoot)
	summary, err := subagent.Run(env.Ctx, env.Cfg, env.Runner.LLM, prompt, func(reg *tool.Registry) {
		register.ExploreTools(reg, env.Cfg, env.Runner.Perm, gi, env.Cfg.LLM.StrictTools)
	}, sub, run, nil)
	status := subagentstore.StatusDone
	errMsg := ""
	if err != nil {
		status = subagentstore.StatusError
		errMsg = err.Error()
	}
	_ = sub.FinishRun(env.Ctx, run.ID, status, errMsg)
	if err != nil {
		return err
	}
	fmt.Fprintln(env.Out, summary)
	return nil
}

func truncateLabel(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
