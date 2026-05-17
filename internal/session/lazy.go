package session

import (
	"context"
	"sync"
	"time"

	"github.com/hejunqiu/ds-code/internal/llm"
	"github.com/google/uuid"
)

// LazyStore delays persisting new sessions until the first store write that
// requires durable storage (e.g. AppendMessage). CreateSession only allocates an
// in-memory session with a new ID so opening the app does not insert empty rows.
type LazyStore struct {
	inner   Store
	pending map[string]Session
	mu      sync.RWMutex
}

// NewLazyStore wraps inner with lazy session creation.
func NewLazyStore(inner Store) *LazyStore {
	return &LazyStore{
		inner:   inner,
		pending: make(map[string]Session),
	}
}

// DropPending removes an unpersisted session from a LazyStore; no-op for other Store implementations.
func DropPending(store Store, id string) {
	if ls, ok := store.(*LazyStore); ok {
		ls.dropPending(id)
	}
}

// dropPending removes an unpersisted session (e.g. after /clear or /resume).
func (l *LazyStore) dropPending(id string) {
	if id == "" {
		return
	}
	l.mu.Lock()
	delete(l.pending, id)
	l.mu.Unlock()
}

func (l *LazyStore) CreateSession(model, effort, thinking, permMode, runMode string) (Session, error) {
	now := time.Now().UTC()
	sess := Session{
		ID:              uuid.NewString(),
		Model:           model,
		ReasoningEffort: effort,
		ThinkingType:    thinking,
		PermissionMode:  permMode,
		RunMode:         runMode,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	l.mu.Lock()
	l.pending[sess.ID] = sess
	l.mu.Unlock()
	return sess, nil
}

func (l *LazyStore) Create(ctx context.Context, s Session) error {
	l.mu.Lock()
	delete(l.pending, s.ID)
	l.mu.Unlock()
	return l.inner.Create(ctx, s)
}

func (l *LazyStore) Get(ctx context.Context, id string) (Session, error) {
	l.mu.RLock()
	if sess, ok := l.pending[id]; ok {
		l.mu.RUnlock()
		return sess, nil
	}
	l.mu.RUnlock()
	return l.inner.Get(ctx, id)
}

func (l *LazyStore) ListMessages(ctx context.Context, sessionID string) ([]Message, error) {
	l.mu.RLock()
	_, pending := l.pending[sessionID]
	l.mu.RUnlock()
	if pending {
		return nil, nil
	}
	return l.inner.ListMessages(ctx, sessionID)
}

func (l *LazyStore) AppendMessage(ctx context.Context, msg Message) error {
	if err := l.materialize(ctx, msg.SessionID); err != nil {
		return err
	}
	return l.inner.AppendMessage(ctx, msg)
}

func (l *LazyStore) AddUsage(ctx context.Context, sessionID string, u llm.Usage) error {
	if err := l.materialize(ctx, sessionID); err != nil {
		return err
	}
	return l.inner.AddUsage(ctx, sessionID, u)
}

func (l *LazyStore) UpdateSession(ctx context.Context, sessionID string, fn func(*Session) error) error {
	l.mu.Lock()
	if sess, ok := l.pending[sessionID]; ok {
		if err := fn(&sess); err != nil {
			l.mu.Unlock()
			return err
		}
		sess.UpdatedAt = time.Now().UTC()
		l.pending[sessionID] = sess
		l.mu.Unlock()
		return nil
	}
	l.mu.Unlock()
	return l.inner.UpdateSession(ctx, sessionID, fn)
}

func (l *LazyStore) ListSessions(ctx context.Context, limit int) ([]Summary, error) {
	return l.inner.ListSessions(ctx, limit)
}

func (l *LazyStore) materialize(ctx context.Context, id string) error {
	l.mu.Lock()
	sess, ok := l.pending[id]
	if !ok {
		l.mu.Unlock()
		return nil
	}
	delete(l.pending, id)
	l.mu.Unlock()
	return l.inner.Create(ctx, sess)
}
