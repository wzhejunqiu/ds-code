package slashcmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/wzhejunqiu/ds-code/internal/agent"
	"github.com/wzhejunqiu/ds-code/internal/agent/spawn"
	ctxpkg "github.com/wzhejunqiu/ds-code/internal/context"
	"github.com/wzhejunqiu/ds-code/internal/version"
)

func Skill(env *Env, args string) error {
	name := strings.TrimSpace(args)
	if name == "" {
		names, err := ctxpkg.ListSkillNames(env.Cfg.ProjectRoot)
		if err != nil {
			return err
		}
		if len(names) == 0 {
			fmt.Fprintf(env.Out, "No skills found under %s/skills/ or ~/%s/skills/\n", version.UserDataDirName, version.UserDataDirName)
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

	meta, body, err := ctxpkg.LoadSkillWithMeta(env.Cfg.ProjectRoot, name)
	if err != nil {
		return err
	}

	if meta.ContextMode == "fork" {
		if env.Spawn == nil || env.SessionID == nil {
			return fmt.Errorf("skill fork requires an active session and spawn service")
		}
		inv := agent.ToolInvocation{
			SessionID:  *env.SessionID,
			ToolCallID: "skill:" + name,
		}
		if env.Store != nil {
			if sess, err := env.Store.Get(env.Ctx, *env.SessionID); err == nil {
				inv.ParentModel = sess.Model
			}
		}
		out, err := env.Spawn.FromSkill(env.Ctx, inv, name, true)
		if err != nil {
			return err
		}
		fmt.Fprintln(env.Out, out)
		return nil
	}

	env.CtxSvc.ActiveSkill = name
	env.CtxSvc.SkillsText = body
	fmt.Fprintf(env.Out, "Activated skill %q (%d chars) for next requests.\n", name, len(body))
	return nil
}

// Ensure spawn.Service implements SpawnRunner at compile time.
var _ SpawnRunner = (*spawn.Service)(nil)

// IsSkillNotFork reports whether err is ErrSkillNotFork.
func IsSkillNotFork(err error) bool {
	return errors.Is(err, spawn.ErrSkillNotFork)
}
