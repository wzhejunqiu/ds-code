package subagent

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/agent"
	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/logging"
	"github.com/wzhejunqiu/ds-code/internal/session"
	"github.com/wzhejunqiu/ds-code/internal/session/subagentstore"
	"go.uber.org/zap"
)

const sessionTitleTimeout = 30 * time.Second

type titleGenState struct {
	wg sync.WaitGroup
}

var titleGenInFlight sync.Map

// SessionTitleWaitTimeout is the recommended wait budget for callers (e.g. non-interactive -p).
const SessionTitleWaitTimeout = sessionTitleTimeout + 5*time.Second

// NewSessionTitleHook returns a Runner hook that asynchronously generates session titles.
func NewSessionTitleHook(cfg *config.Config, llmClient llm.Client, store session.Store, subStore subagentstore.Store) agent.SessionTitleHook {
	if cfg == nil || llmClient == nil || store == nil || subStore == nil {
		return nil
	}
	return func(_ context.Context, sessionID, userContent string) {
		state := &titleGenState{}
		state.wg.Add(1)
		if _, loaded := titleGenInFlight.LoadOrStore(sessionID, state); loaded {
			state.wg.Done()
			return
		}
		go func() {
			defer titleGenInFlight.Delete(sessionID)
			defer state.wg.Done()
			generateSessionTitleAsync(cfg, llmClient, store, subStore, sessionID, userContent)
		}()
	}
}

// WaitForSessionTitle blocks until in-flight title generation for sessionID finishes or timeout elapses.
func WaitForSessionTitle(sessionID string, timeout time.Duration) {
	v, ok := titleGenInFlight.Load(sessionID)
	if !ok {
		return
	}
	state, ok := v.(*titleGenState)
	if !ok {
		return
	}
	done := make(chan struct{})
	go func() {
		state.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(timeout):
	}
}

func generateSessionTitleAsync(cfg *config.Config, llmClient llm.Client, store session.Store, subStore subagentstore.Store, sessionID, userContent string) {
	ctx, cancel := context.WithTimeout(context.Background(), sessionTitleTimeout)
	defer cancel()

	title, err := GenerateSessionTitle(ctx, cfg, llmClient, subStore, sessionID, userContent)
	if err != nil {
		logging.L().Debug("session title subagent failed",
			zap.String("session_id", sessionID),
			zap.Error(err),
		)
		return
	}
	if strings.TrimSpace(title) == "" {
		return
	}
	if err := store.UpdateSession(ctx, sessionID, func(s *session.Session) error {
		s.Title = title
		return nil
	}); err != nil {
		logging.L().Debug("session title update failed",
			zap.String("session_id", sessionID),
			zap.Error(err),
		)
	}
}
