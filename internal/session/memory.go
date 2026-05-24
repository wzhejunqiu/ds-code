package session

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/role"
)

// MemoryStore is an in-memory session.Store for Phase 1–2.
type MemoryStore struct {
	mu       sync.RWMutex
	sessions map[string]Session
	messages map[string][]Message
	nextID   int64
}

// NewMemoryStore creates an empty store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		sessions: make(map[string]Session),
		messages: make(map[string][]Message),
	}
}

// CreateSession implements Store.
func (m *MemoryStore) CreateSession(model, effort, thinking, permMode, runMode string) (Session, error) {
	return m.newSession(model, effort, thinking, permMode, runMode)
}

// NewSession creates a session with defaults from config fields.
func (m *MemoryStore) NewSession(model, effort, thinking, permMode, runMode string) (Session, error) {
	return m.newSession(model, effort, thinking, permMode, runMode)
}

func (m *MemoryStore) newSession(model, effort, thinking, permMode, runMode string) (Session, error) {
	now := time.Now().UTC()
	s := Session{
		ID:              uuid.NewString(),
		Model:           model,
		ReasoningEffort: effort,
		ThinkingType:    thinking,
		PermissionMode:  permMode,
		RunMode:         runMode,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := m.Create(context.Background(), s); err != nil {
		return Session{}, err
	}
	return s, nil
}

func (m *MemoryStore) Create(_ context.Context, s Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.sessions[s.ID]; ok {
		return fmt.Errorf("session: already exists %s", s.ID)
	}
	m.sessions[s.ID] = s
	m.messages[s.ID] = nil
	return nil
}

func (m *MemoryStore) Get(_ context.Context, id string) (Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[id]
	if !ok {
		return Session{}, fmt.Errorf("session: not found %s", id)
	}
	return s, nil
}

func (m *MemoryStore) ListMessages(_ context.Context, sessionID string) ([]Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	msgs, ok := m.messages[sessionID]
	if !ok {
		return nil, fmt.Errorf("session: not found %s", sessionID)
	}
	out := make([]Message, len(msgs))
	copy(out, msgs)
	return out, nil
}

func (m *MemoryStore) ListSessions(_ context.Context, limit int) ([]Summary, error) {
	if limit <= 0 {
		limit = 50
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	type item struct {
		s Summary
		t time.Time
	}
	var items []item
	for id, sess := range m.sessions {
		if len(m.messages[id]) == 0 {
			continue
		}
		title := sess.Title
		if title == "" {
			title = "(untitled)"
		}
		items = append(items, item{
			s: Summary{
				ID:           id,
				Title:        title,
				Model:        sess.Model,
				BilledTokens: BilledTokens(sess),
				UpdatedAt:    sess.UpdatedAt,
				CreatedAt:    sess.CreatedAt,
			},
			t: sess.UpdatedAt,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].t.After(items[j].t)
	})
	if len(items) > limit {
		items = items[:limit]
	}
	out := make([]Summary, len(items))
	for i, it := range items {
		out[i] = it.s
	}
	return out, nil
}

func (m *MemoryStore) AppendMessage(_ context.Context, msg Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.sessions[msg.SessionID]; !ok {
		return fmt.Errorf("session: not found %s", msg.SessionID)
	}
	m.nextID++
	msg.ID = m.nextID
	msg.CreatedAt = time.Now().UTC()
	m.messages[msg.SessionID] = append(m.messages[msg.SessionID], msg)
	LogAppendDebug(msg)
	s := m.sessions[msg.SessionID]
	s.UpdatedAt = msg.CreatedAt
	if msg.Role == role.User && s.Title == "" {
		s.Title = TruncateTitle(msg.Content, MaxTitleRunes)
	}
	if msg.EstimatedCostCNY > 0 {
		s.EstimatedCostCNY += msg.EstimatedCostCNY
	}
	m.sessions[msg.SessionID] = s
	return nil
}

func (m *MemoryStore) UpdateSession(_ context.Context, sessionID string, fn func(*Session) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[sessionID]
	if !ok {
		return fmt.Errorf("session: not found %s", sessionID)
	}
	if err := fn(&s); err != nil {
		return err
	}
	s.UpdatedAt = time.Now().UTC()
	m.sessions[sessionID] = s
	return nil
}

func (m *MemoryStore) AddUsage(_ context.Context, sessionID string, u llm.Usage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[sessionID]
	if !ok {
		return fmt.Errorf("session: not found %s", sessionID)
	}
	s.PromptTokensTotal += int64(u.PromptTokens)
	s.CompletionTokensTotal += int64(u.CompletionTokens)
	s.PromptCacheHitTokensTotal += int64(u.PromptCacheHitTokens)
	s.UpdatedAt = time.Now().UTC()
	m.sessions[sessionID] = s
	LogAddUsageDebug(sessionID, u)
	return nil
}
