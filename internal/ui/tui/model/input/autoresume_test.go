package input_test

import (
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/agent"
	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/role"
	"github.com/wzhejunqiu/ds-code/internal/session"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/deps"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/input"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/state"
)

func TestTryAutoResumeTurn_idleWithPendingNotification(t *testing.T) {
	store := session.NewMemoryStore()
	sess, err := store.NewSession("m", "max", "enabled", "auto", "agent")
	if err != nil {
		t.Fatal(err)
	}
	pending := true
	s := &state.State{
		SessionID: sess.ID,
		Deps: &deps.Deps{
			Store:  store,
			Runner: &agent.Runner{},
			HasPendingNotifications: func() bool {
				return pending
			},
			BackgroundAgents: func() int { return 0 },
		},
	}
	cmd := input.TryAutoResumeTurn(s, func() {}, func() {})
	if cmd == nil {
		t.Fatal("expected auto-resume command")
	}
	if !s.Running {
		t.Fatal("expected Running=true after SubmitAutoResume path")
	}
}

func TestTryAutoResumeTurn_blockedWhileRunning(t *testing.T) {
	s := &state.State{
		Running: true,
		Deps: &deps.Deps{
			HasPendingNotifications: func() bool { return true },
		},
	}
	if cmd := input.TryAutoResumeTurn(s, func() {}, func() {}); cmd != nil {
		t.Fatal("expected nil while main turn running")
	}
}

func TestTryAutoResumeTurn_sessionTailNotification(t *testing.T) {
	store := session.NewMemoryStore()
	sess, err := store.NewSession("m", "max", "enabled", "auto", "agent")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.AppendMessage(t.Context(), session.Message{
		SessionID: sess.ID,
		Role:      role.User,
		Content:   "<task-notification>\n  <task-id>sa-1</task-id>\n  <status>completed</status>\n</task-notification>",
	}); err != nil {
		t.Fatal(err)
	}
	s := &state.State{
		SessionID: sess.ID,
		Deps: &deps.Deps{
			Store:                   store,
			Runner:                  &agent.Runner{},
			HasPendingNotifications: func() bool { return false },
			BackgroundAgents:        func() int { return 0 },
		},
	}
	if cmd := input.TryAutoResumeTurn(s, func() {}, func() {}); cmd == nil {
		t.Fatal("expected auto-resume for session tail notification")
	}
}

func TestTryAutoResumeTurn_blockedWhileBackgroundAgentsRunning(t *testing.T) {
	s := &state.State{
		Deps: &deps.Deps{
			HasPendingNotifications: func() bool { return true },
			BackgroundAgents:        func() int { return 1 },
			Runner:                  &agent.Runner{},
		},
	}
	if cmd := input.TryAutoResumeTurn(s, func() {}, func() {}); cmd != nil {
		t.Fatal("expected nil while background agents still running")
	}
}

func TestTryAutoResumeTurn_noTriggerWithoutSignal(t *testing.T) {
	store := session.NewMemoryStore()
	sess, err := store.NewSession("m", "max", "enabled", "auto", "agent")
	if err != nil {
		t.Fatal(err)
	}
	s := &state.State{
		SessionID: sess.ID,
		Deps: &deps.Deps{
			Store:                   store,
			Runner:                  &agent.Runner{},
			Cfg:                     &config.Config{},
			HasPendingNotifications: func() bool { return false },
			BackgroundAgents:        func() int { return 0 },
		},
	}
	if cmd := input.TryAutoResumeTurn(s, func() {}, func() {}); cmd != nil {
		t.Fatal("expected nil without pending notification or session tail")
	}
}
