package slashcmd

import (
	"fmt"
	"strings"

	"github.com/hejunqiu/ds-code/internal/agent/subagent"
	"github.com/hejunqiu/ds-code/internal/tool"
	"github.com/hejunqiu/ds-code/internal/tool/builtin"
)

func Task(env *Env, args string) error {
	prompt := strings.TrimSpace(args)
	if prompt == "" {
		return fmt.Errorf("usage: /task <prompt>")
	}
	fmt.Fprintln(env.Out, "Running read-only subagent...")
	gi, _ := tool.LoadGitignore(env.Cfg.ProjectRoot)
	summary, err := subagent.Run(env.Ctx, env.Cfg, env.Runner.LLM, prompt, func(reg *tool.Registry) {
		builtin.RegisterExploreTools(reg, env.Cfg, env.Runner.Perm, gi, env.Cfg.LLM.StrictTools)
	})
	if err != nil {
		return err
	}
	fmt.Fprintln(env.Out, summary)
	return nil
}
