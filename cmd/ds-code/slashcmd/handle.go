package slashcmd

import (
	"fmt"

	"github.com/hejunqiu/ds-code/internal/session"
	uislash "github.com/hejunqiu/ds-code/internal/ui/slash"
)

// Handle parses and dispatches a slash command line.
func Handle(env *Env, host Host, line string) (handled bool, err error) {
	cmd, args, ok := uislash.Parse(line)
	if !ok {
		return false, nil
	}

	switch cmd {
	case "help":
		uislash.WriteHelp(env.Out)
		return true, nil

	case "git":
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
		return true, Mode(env, args)

	case "effort":
		return true, Effort(env, args)

	case "thinking":
		return true, Thinking(env, args)

	case "permissions":
		return true, Permissions(env, args)

	case "clear":
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
		if err := env.CtxSvc.CompactAPIContext(env.Ctx, *env.SessionID); err != nil {
			return true, err
		}
		fmt.Fprintln(env.Out, "API context compacted (history rows unchanged).")
		return true, nil

	case "context":
		return true, Context(env, args)

	case "resume":
		return true, Resume(env, args)

	case "plan":
		return true, host.SetRunMode(env.Ctx, env, "plan")

	case "agent":
		return true, host.SetRunMode(env.Ctx, env, "agent")

	case "skill":
		return true, Skill(env, args)

	case "task":
		return true, Task(env, args)

	case "checkpoint":
		return true, Checkpoint(env, args)

	case "rewind":
		return true, Rewind(env, args)

	case "btw":
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
