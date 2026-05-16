package main

import (
	"fmt"
	"strings"

	"github.com/hejunqiu/ds-code/internal/agent/subagent"
	ctxpkg "github.com/hejunqiu/ds-code/internal/context"
	"github.com/hejunqiu/ds-code/internal/session"
	"github.com/hejunqiu/ds-code/internal/tool"
	"github.com/hejunqiu/ds-code/internal/tool/builtin"
)

func (a *app) slashRunMode(env *slashEnv, mode string) error {
	a.cfg.RunMode = mode
	if err := env.store.UpdateSession(env.ctx, *env.sessionID, func(s *session.Session) error {
		s.RunMode = mode
		return nil
	}); err != nil {
		return err
	}
	gi, _ := tool.LoadGitignore(a.cfg.ProjectRoot)
	bundle, err := a.buildTools(env.ctx, env.runner.Perm, gi, a.cfg.LLM.StrictTools, env.runner.LLM, mode)
	if err != nil {
		return err
	}
	a.rebindRunnerTools(env.runner, env.ctxSvc, bundle)
	fmt.Fprintf(env.out, "Run mode set to %s (tools updated for this session).\n", mode)
	return nil
}

func (a *app) slashSkill(env *slashEnv, args string) error {
	name := strings.TrimSpace(args)
	if name == "" {
		names, err := ctxpkg.ListSkillNames(a.cfg.ProjectRoot)
		if err != nil {
			return err
		}
		if len(names) == 0 {
			fmt.Fprintln(env.out, "No skills found under .ds-code/skills/ or ~/.ds-code/skills/")
			return nil
		}
		fmt.Fprintln(env.out, "Available skills:")
		for _, n := range names {
			mark := ""
			if n == env.ctxSvc.ActiveSkill {
				mark = " (active)"
			}
			fmt.Fprintf(env.out, "  %s%s\n", n, mark)
		}
		return nil
	}
	text, err := ctxpkg.LoadSkill(a.cfg.ProjectRoot, name)
	if err != nil {
		return err
	}
	env.ctxSvc.ActiveSkill = name
	env.ctxSvc.SkillsText = text
	fmt.Fprintf(env.out, "Activated skill %q (%d chars) for next requests.\n", name, len(text))
	return nil
}

func (a *app) slashTask(env *slashEnv, args string) error {
	prompt := strings.TrimSpace(args)
	if prompt == "" {
		return fmt.Errorf("usage: /task <prompt>")
	}
	fmt.Fprintln(env.out, "Running read-only subagent...")
	gi, _ := tool.LoadGitignore(a.cfg.ProjectRoot)
	summary, err := subagent.Run(env.ctx, a.cfg, env.runner.LLM, prompt, func(reg *tool.Registry) {
		builtin.RegisterExploreTools(reg, a.cfg, env.runner.Perm, gi, a.cfg.LLM.StrictTools)
	})
	if err != nil {
		return err
	}
	fmt.Fprintln(env.out, summary)
	return nil
}
