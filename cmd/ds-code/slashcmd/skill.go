package slashcmd

import (
	"fmt"
	"strings"

	ctxpkg "github.com/hejunqiu/ds-code/internal/context"
)

func Skill(env *Env, args string) error {
	name := strings.TrimSpace(args)
	if name == "" {
		names, err := ctxpkg.ListSkillNames(env.Cfg.ProjectRoot)
		if err != nil {
			return err
		}
		if len(names) == 0 {
			fmt.Fprintln(env.Out, "No skills found under .ds-code/skills/ or ~/.ds-code/skills/")
			return nil
		}
		fmt.Fprintln(env.Out, "Available skills:")
		for _, n := range names {
			mark := ""
			if n == env.CtxSvc.ActiveSkill {
				mark = " (active)"
			}
			fmt.Fprintf(env.Out, "  %s%s\n", n, mark)
		}
		return nil
	}
	text, err := ctxpkg.LoadSkill(env.Cfg.ProjectRoot, name)
	if err != nil {
		return err
	}
	env.CtxSvc.ActiveSkill = name
	env.CtxSvc.SkillsText = text
	fmt.Fprintf(env.Out, "Activated skill %q (%d chars) for next requests.\n", name, len(text))
	return nil
}
