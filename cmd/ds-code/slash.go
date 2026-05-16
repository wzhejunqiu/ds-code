package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/hejunqiu/ds-code/internal/agent"
	ctxpkg "github.com/hejunqiu/ds-code/internal/context"
	"github.com/hejunqiu/ds-code/internal/permission"
	"github.com/hejunqiu/ds-code/internal/session"
	uipkg "github.com/hejunqiu/ds-code/internal/ui"
	"github.com/hejunqiu/ds-code/internal/ui/slash"
)

type slashEnv struct {
	ctx       context.Context
	out       io.Writer
	cfg       *app
	runner    *agent.Runner
	store     session.Store
	ctxSvc    *ctxpkg.Service
	sessionID *string
}

func (a *app) handleSlash(env *slashEnv, line string) (handled bool, err error) {
	cmd, args, ok := slash.Parse(line)
	if !ok {
		return false, nil
	}

	switch cmd {
	case "help":
		slash.WriteHelp(env.out)
		return true, nil

	case "git":
		snap, err := env.ctxSvc.RefreshGitSnapshot(env.ctx, *env.sessionID, a.cfg.ProjectRoot)
		if err != nil {
			return true, err
		}
		if snap == "" {
			fmt.Fprintln(env.out, "Not a git repository (or git unavailable); snapshot cleared.")
		} else {
			fmt.Fprintf(env.out, "Git snapshot updated (%d chars) for next request.\n", len(snap))
		}
		return true, nil

	case "mode":
		return true, a.slashMode(env, args)

	case "effort":
		return true, a.slashEffort(env, args)

	case "thinking":
		return true, a.slashThinking(env, args)

	case "permissions":
		return true, a.slashPermissions(env, args)

	case "clear":
		sess, err := a.createSession(env.store)
		if err != nil {
			return true, err
		}
		if err := a.seedGitSnapshot(env.ctx, env.store, env.ctxSvc, sess.ID); err != nil {
			return true, err
		}
		*env.sessionID = sess.ID
		fmt.Fprintf(env.out, "New session: %s\n", sess.ID)
		return true, nil

	case "compact":
		if err := env.ctxSvc.CompactAPIContext(env.ctx, *env.sessionID); err != nil {
			return true, err
		}
		fmt.Fprintln(env.out, "API context compacted (history rows unchanged).")
		return true, nil

	case "context":
		return true, a.slashContext(env, args)

	case "resume":
		return true, a.slashResume(env, args)

	default:
		if c, ok := slash.Lookup(cmd); ok && c.Phase != "" {
			fmt.Fprintf(env.out, "/%s is planned for Phase %s.\n", cmd, c.Phase)
			return true, nil
		}
		fmt.Fprintf(env.out, "Unknown command: /%s (try /help)\n", cmd)
		return true, nil
	}
}

func (a *app) slashContext(env *slashEnv, args string) error {
	view, err := env.ctxSvc.BuildAPIContext(env.ctx, *env.sessionID)
	if err != nil {
		return err
	}
	sess, err := env.store.Get(env.ctx, *env.sessionID)
	if err != nil {
		return err
	}
	panel, err := uipkg.BuildContextPanelData(a.cfg, sess, view)
	if err != nil {
		return err
	}
	if strings.TrimSpace(args) == "--json" {
		text, err := uipkg.FormatContextJSON(panel)
		if err != nil {
			return err
		}
		fmt.Fprintln(env.out, text)
		return nil
	}
	fmt.Fprintln(env.out, uipkg.FormatContextPanel(panel))
	return nil
}

func (a *app) slashMode(env *slashEnv, args string) error {
	return env.store.UpdateSession(env.ctx, *env.sessionID, func(s *session.Session) error {
		if args == "" {
			fmt.Fprintf(env.out, "model: %s\n", s.Model)
			return nil
		}
		switch args {
		case "deepseek-v4-pro", "deepseek-v4-flash":
			s.Model = args
			fmt.Fprintf(env.out, "model set to %s\n", args)
		default:
			return fmt.Errorf("invalid model %q", args)
		}
		return nil
	})
}

func (a *app) slashEffort(env *slashEnv, args string) error {
	return env.store.UpdateSession(env.ctx, *env.sessionID, func(s *session.Session) error {
		if args == "" {
			fmt.Fprintf(env.out, "reasoning_effort: %s\n", s.ReasoningEffort)
			return nil
		}
		switch args {
		case "high", "max":
			s.ReasoningEffort = args
			fmt.Fprintf(env.out, "reasoning_effort set to %s\n", args)
		default:
			return fmt.Errorf("invalid effort %q", args)
		}
		return nil
	})
}

func (a *app) slashThinking(env *slashEnv, args string) error {
	return env.store.UpdateSession(env.ctx, *env.sessionID, func(s *session.Session) error {
		if args == "" {
			fmt.Fprintf(env.out, "thinking: %s\n", s.ThinkingType)
			return nil
		}
		switch strings.ToLower(args) {
		case "on", "enabled":
			s.ThinkingType = "enabled"
			fmt.Fprintln(env.out, "thinking enabled")
		case "off", "disabled":
			s.ThinkingType = "disabled"
			fmt.Fprintln(env.out, "thinking disabled")
		default:
			return fmt.Errorf("invalid thinking %q (use on|off)", args)
		}
		return nil
	})
}

func (a *app) slashPermissions(env *slashEnv, args string) error {
	if args == "" {
		sess, err := env.store.Get(env.ctx, *env.sessionID)
		if err != nil {
			return err
		}
		fmt.Fprintf(env.out, "permission.mode: %s (session), runner: %s\n", sess.PermissionMode, env.runner.Perm.Mode)
		return nil
	}
	switch args {
	case "readonly", "ask", "auto":
		a.cfg.Permission.Mode = args
		env.runner.Perm.Mode = args
		if args == "ask" && permission.IsInteractiveTTY() {
			env.runner.Perm.Prompter = permission.StdinPrompter(os.Stderr)
		} else {
			env.runner.Perm.Prompter = nil
		}
		_ = env.store.UpdateSession(env.ctx, *env.sessionID, func(s *session.Session) error {
			s.PermissionMode = args
			return nil
		})
		fmt.Fprintf(env.out, "permission.mode set to %s\n", args)
	default:
		return fmt.Errorf("invalid permission mode %q", args)
	}
	return nil
}

func (a *app) slashResume(env *slashEnv, args string) error {
	id := strings.TrimSpace(args)
	if id == "" {
		list, err := env.store.ListSessions(env.ctx, 10)
		if err != nil {
			return err
		}
		if len(list) == 0 {
			fmt.Fprintln(env.out, "No saved sessions.")
			return nil
		}
		fmt.Fprintln(env.out, "Recent sessions (use /resume <id>):")
		for _, s := range list {
			fmt.Fprintf(env.out, "  %s  %s\n", s.ID, s.Title)
		}
		return nil
	}
	if _, err := env.store.Get(env.ctx, id); err != nil {
		return err
	}
	*env.sessionID = id
	fmt.Fprintf(env.out, "Resumed session %s\n", id)
	return nil
}

func (a *app) seedGitSnapshot(ctx context.Context, store session.Store, ctxSvc *ctxpkg.Service, sessionID string) error {
	snap, err := ctxpkg.CaptureGitSnapshot(a.cfg.ProjectRoot, a.cfg.Context.GitSnapshotMaxChars)
	if err != nil || snap == "" {
		return err
	}
	return store.UpdateSession(ctx, sessionID, func(s *session.Session) error {
		s.GitSnapshot = snap
		return nil
	})
}
