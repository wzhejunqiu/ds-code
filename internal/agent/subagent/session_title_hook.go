package subagent

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/hejunqiu/ds-code/internal/agent"
	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/llm"
	"github.com/hejunqiu/ds-code/internal/logging"
	"github.com/hejunqiu/ds-code/internal/session"
	"github.com/hejunqiu/ds-code/internal/session/subagentstore"
	"go.uber.org/zap"
)

const sessionTitleTimeout = 30 * time.Second

var titleGenInFlight sync.Map

// NewSessionTitleHook returns a Runner hook that asynchronously generates session titles.
func NewSessionTitleHook(cfg *config.Config, llmClient llm.Client, store session.Store, subStore subagentstore.Store) agent.SessionTitleHook {
	if cfg == nil || llmClient == nil || store == nil || subStore == nil {
		return nil
	}
	return func(parentCtx context.Context, sessionID, userContent string) {
		if _, loaded := titleGenInFlight.LoadOrStore(sessionID, struct{}{}); loaded {
			return
		}
		go func() {
			defer titleGenInFlight.Delete(sessionID)
			generateSessionTitleAsync(parentCtx, cfg, llmClient, store, subStore, sessionID, userContent)
		}()
	}
}

func generateSessionTitleAsync(parentCtx context.Context, cfg *config.Config, llmClient llm.Client, store session.Store, subStore subagentstore.Store, sessionID, userContent string) {
	ctx, cancel := context.WithTimeout(parentCtx, sessionTitleTimeout)
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
