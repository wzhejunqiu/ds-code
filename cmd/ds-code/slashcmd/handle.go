package slashcmd

import (
	"fmt"

	"github.com/wzhejunqiu/ds-code/internal/session"
	uislash "github.com/wzhejunqiu/ds-code/internal/ui/slash"
)

// Handle parses and dispatches a slash command line.
func Handle(env *Env, host Host, line string) (handled bool, err error) {
	cmd, args, ok := uislash.Parse(line)
	if !ok {
		return false, nil
	}
	if env == nil {
		return true, fmt.Errorf("slashcmd: nil env")
	}
	if env.Out == nil {
		return true, fmt.Errorf("slashcmd: nil output writer")
	}

	switch cmd {
	case "help":
		uislash.WriteHelp(env.Out)
		return true, nil

	case "git":
		if err := requireSessionEnv(env); err != nil {
			return true, err
		}
		if env.CtxSvc == nil {
			return true, fmt.Errorf("slashcmd: nil context service")
		}
		snap, err := env.CtxSvc.RefreshGitSnapshot(env.Ctx, *env.SessionID, env.Cfg.ProjectRoot)
		if err != nil {
			return true, err
		}
		if snap == "" {
			fmt.Fprintln(env.Out, "Not a git repository (or git unavailable); snapshot cleared.")
		} else {
			fmt.Fprintf(env.Out, "Git snapshot updated (%d chars) for next request.\n", len(snap))
		}
		return true, nil

	case "mode":
		if err := requireSessionEnv(env); err != nil {
			return true, err
		}
		return true, Mode(env, args)

	case "effort":
		if err := requireSessionEnv(env); err != nil {
			return true, err
		}
		return true, Effort(env, args)

	case "thinking":
		if err := requireSessionEnv(env); err != nil {
			return true, err
		}
		return true, Thinking(env, args)

	case "permissions":
		if err := requireSessionEnv(env); err != nil {
			return true, err
		}
		return true, Permissions(env, args)

	case "clear":
		if err := requireSessionEnv(env); err != nil {
			return true, err
		}
		session.DropPending(env.Store, *env.SessionID)
		sess, err := CreateSession(env.Cfg, env.Store)
		if err != nil {
			return true, err
		}
		if err := SeedGitSnapshot(env.Cfg, env.Ctx, env.Store, sess.ID); err != nil {
			return true, err
		}
		*env.SessionID = sess.ID
		fmt.Fprintf(env.Out, "New session: %s\n", sess.ID)
		return true, nil

	case "compact":
		if err := requireSessionEnv(env); err != nil {
			return true, err
		}
		if env.CtxSvc == nil {
			return true, fmt.Errorf("slashcmd: nil context service")
		}
		if err := env.CtxSvc.CompactAPIContext(env.Ctx, *env.SessionID); err != nil {
			return true, err
		}
		fmt.Fprintln(env.Out, "API context compacted (history rows unchanged).")
		return true, nil

	case "context":
		if err := requireSessionEnv(env); err != nil {
			return true, err
		}
		if env.CtxSvc == nil {
			return true, fmt.Errorf("slashcmd: nil context service")
		}
		return true, Context(env, args)

	case "resume":
		if err := requireSessionEnv(env); err != nil {
			return true, err
		}
		return true, Resume(env, args)

	case "plan":
		if err := requireSessionEnv(env); err != nil {
			return true, err
		}
		if host == nil {
			return true, fmt.Errorf("slashcmd: nil host")
		}
		return true, host.SetRunMode(env.Ctx, env, "plan")

	case "agent":
		if err := requireSessionEnv(env); err != nil {
			return true, err
		}
		if host == nil {
			return true, fmt.Errorf("slashcmd: nil host")
		}
		return true, host.SetRunMode(env.Ctx, env, "agent")

	case "skill":
		if err := requireSessionEnv(env); err != nil {
			return true, err
		}
		return true, Skill(env, args)

	case "remember":
		if err := requireSessionEnv(env); err != nil {
			return true, err
		}
		return true, Remember(env, args)

	case "task":
		if err := requireSessionEnv(env); err != nil {
			return true, err
		}
		return true, Agent(env, args)

	case "checkpoint":
		if err := requireSessionEnv(env); err != nil {
			return true, err
		}
		return true, Checkpoint(env, args)

	case "rewind":
		if err := requireSessionEnv(env); err != nil {
			return true, err
		}
		return true, Rewind(env, args)

	case "btw":
		if err := requireSessionEnv(env); err != nil {
			return true, err
		}
		if env.Runner == nil {
			return true, fmt.Errorf("slashcmd: nil runner")
		}
		return true, Btw(env, args)

	default:
		if c, ok := uislash.Lookup(cmd); ok && c.Phase != "" {
			fmt.Fprintf(env.Out, "/%s is planned for Phase %s.\n", cmd, c.Phase)
			return true, nil
		}
		fmt.Fprintf(env.Out, "Unknown command: /%s (try /help)\n", cmd)
		return true, nil
	}
}
