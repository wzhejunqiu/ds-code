package slashcmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/hejunqiu/ds-code/internal/config"
	ctxpkg "github.com/hejunqiu/ds-code/internal/context"
	"github.com/hejunqiu/ds-code/internal/logging"
	"github.com/hejunqiu/ds-code/internal/permission"
	"github.com/hejunqiu/ds-code/internal/session"
	"go.uber.org/zap"
)

// CreateSession opens a new session using defaults from cfg.
func CreateSession(cfg *config.Config, store session.Store) (session.Session, error) {
	return store.CreateSession(
		cfg.LLM.Model,
		cfg.LLM.ReasoningEffort,
		cfg.LLM.Thinking.Type,
		cfg.Permission.Mode,
		cfg.RunMode,
	)
}

// SeedGitSnapshot captures git state into the session row when in a git repo.
func SeedGitSnapshot(cfg *config.Config, ctx context.Context, store session.Store, sessionID string) error {
	snap, err := ctxpkg.CaptureGitSnapshot(cfg.ProjectRoot, cfg.Context.GitSnapshotMaxChars)
	if err != nil || snap == "" {
		return err
	}
	logging.L().Debug("git snapshot seeded", zap.String("session_id", sessionID), zap.Int("chars", len(snap)))
	return store.UpdateSession(ctx, sessionID, func(s *session.Session) error {
		s.GitSnapshot = snap
		return nil
	})
}

func Mode(env *Env, args string) error {
	return env.Store.UpdateSession(env.Ctx, *env.SessionID, func(s *session.Session) error {
		if args == "" {
			fmt.Fprintf(env.Out, "model: %s\n", s.Model)
			return nil
		}
		switch args {
		case "deepseek-v4-pro", "deepseek-v4-flash":
			s.Model = args
			fmt.Fprintf(env.Out, "model set to %s\n", args)
		default:
			return fmt.Errorf("invalid model %q", args)
		}
		return nil
	})
}

func Effort(env *Env, args string) error {
	return env.Store.UpdateSession(env.Ctx, *env.SessionID, func(s *session.Session) error {
		if args == "" {
			fmt.Fprintf(env.Out, "reasoning_effort: %s\n", s.ReasoningEffort)
			return nil
		}
		switch args {
		case "high", "max":
			s.ReasoningEffort = args
			fmt.Fprintf(env.Out, "reasoning_effort set to %s\n", args)
		default:
			return fmt.Errorf("invalid effort %q", args)
		}
		return nil
	})
}

func Thinking(env *Env, args string) error {
	return env.Store.UpdateSession(env.Ctx, *env.SessionID, func(s *session.Session) error {
		if args == "" {
			fmt.Fprintf(env.Out, "thinking: %s\n", s.ThinkingType)
			return nil
		}
		switch strings.ToLower(args) {
		case "on", "enabled":
			s.ThinkingType = "enabled"
			fmt.Fprintln(env.Out, "thinking enabled")
		case "off", "disabled":
			s.ThinkingType = "disabled"
			fmt.Fprintln(env.Out, "thinking disabled")
		default:
			return fmt.Errorf("invalid thinking %q (use on|off)", args)
		}
		return nil
	})
}

func Permissions(env *Env, args string) error {
	if args == "" {
		sess, err := env.Store.Get(env.Ctx, *env.SessionID)
		if err != nil {
			return err
		}
		fmt.Fprintf(env.Out, "permission.mode: %s (session), runner: %s\n", sess.PermissionMode, env.Runner.Perm.Mode)
		return nil
	}
	mode, confirmed, err := parsePermissionArgs(args)
	if err != nil {
		return err
	}
	switch mode {
	case "readonly", "ask", "auto":
		if mode == "auto" && !confirmed {
			fmt.Fprintf(env.Out, "Setting permission.mode to auto runs write/shell tools without confirmation. Re-run with: /permissions auto --yes\n")
			return nil
		}
		env.Cfg.Permission.Mode = mode
		env.Runner.Perm.Mode = mode
		if mode == "ask" && permission.IsInteractiveTTY() {
			env.Runner.Perm.Prompter = permission.StdinPrompter(os.Stderr)
		} else {
			env.Runner.Perm.Prompter = nil
		}
		_ = env.Store.UpdateSession(env.Ctx, *env.SessionID, func(s *session.Session) error {
			s.PermissionMode = mode
			return nil
		})
		fmt.Fprintf(env.Out, "permission.mode set to %s\n", mode)
	default:
		return fmt.Errorf("invalid permission mode %q", mode)
	}
	return nil
}

func parsePermissionArgs(raw string) (mode string, confirmed bool, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false, fmt.Errorf("usage: /permissions [readonly|ask|auto [--yes]]")
	}
	if i := strings.LastIndex(raw, " --yes"); i >= 0 && strings.TrimSpace(raw[i:]) == "--yes" {
		confirmed = true
		raw = strings.TrimSpace(raw[:i])
	}
	switch raw {
	case "readonly", "ask", "auto":
		return raw, confirmed, nil
	default:
		return "", false, fmt.Errorf("invalid permission mode %q", raw)
	}
}

func Resume(env *Env, args string) error {
	id := strings.TrimSpace(args)
	if id == "" {
		list, err := env.Store.ListSessions(env.Ctx, 50)
		if err != nil {
			return err
		}
		if len(list) == 0 {
			fmt.Fprintln(env.Out, "No saved sessions.")
			return nil
		}
		fmt.Fprintln(env.Out, "Recent sessions (use /resume <id>):")
		for _, s := range list {
			fmt.Fprintf(env.Out, "  %s  %s\n", s.ID, s.Title)
		}
		return nil
	}
	if _, err := env.Store.Get(env.Ctx, id); err != nil {
		return err
	}
	session.DropPending(env.Store, *env.SessionID)
	*env.SessionID = id
	fmt.Fprintf(env.Out, "Resumed session %s\n", id)
	return nil
}
