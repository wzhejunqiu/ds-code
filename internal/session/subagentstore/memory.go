package subagentstore

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/llm"
)

// MemoryStore is an in-memory SubagentStore for tests.
type MemoryStore struct {
	mu      sync.Mutex
	runs    map[string]Run
	msgs    map[string][]Message
	nextMsg int64
}

// NewMemoryStore creates an empty in-memory subagent store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		runs: make(map[string]Run),
		msgs: make(map[string][]Message),
	}
}

func (m *MemoryStore) CreateRun(_ context.Context, p CreateRunParams) (Run, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now().UTC()
	id := fmt.Sprintf("sa-%d", len(m.runs)+1)
	kind := DefaultRunKind(p.RunKind)
	r := Run{
		ID:                  id,
		ParentSessionID:     p.ParentSessionID,
		ParentToolCallID:    p.ParentToolCallID,
		RunKind:             kind,
		Label:               p.Label,
		Prompt:              p.Prompt,
		Status:              StatusRunning,
		Model:               p.Model,
		ReasoningEffort:     p.ReasoningEffort,
		ThinkingType:        p.ThinkingType,
		PricingSnapshotJSON: p.PricingSnapshotJSON,
		CreatedAt:           now,
	}
	m.runs[id] = r
	return r, nil
}

func (m *MemoryStore) FinishRun(_ context.Context, runID string, status Status, errMsg string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.runs[runID]
	if !ok {
		return fmt.Errorf("subagent run not found: %s", runID)
	}
	r.Status = status
	r.Error = errMsg
	r.EndedAt = time.Now().UTC()
	var cost float64
	for _, msg := range m.msgs[runID] {
		cost += msg.EstimatedCostCNY
	}
	r.EstimatedCostCNY = cost
	m.runs[runID] = r
	return nil
}

func (m *MemoryStore) ListRuns(_ context.Context, parentSessionID string) ([]Run, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []Run
	for _, r := range m.runs {
		if r.ParentSessionID == parentSessionID {
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out, nil
}

func (m *MemoryStore) GetRun(_ context.Context, runID string) (Run, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.runs[runID]
	if !ok {
		return Run{}, fmt.Errorf("subagent run not found: %s", runID)
	}
	return r, nil
}

func (m *MemoryStore) GetRunByToolCall(_ context.Context, parentSessionID, parentToolCallID string) (Run, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range m.runs {
		if r.ParentSessionID == parentSessionID && r.ParentToolCallID == parentToolCallID {
			return r, nil
		}
	}
	return Run{}, fmt.Errorf("subagent run not found for tool call %s", parentToolCallID)
}

func (m *MemoryStore) AppendMessage(_ context.Context, msg Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.runs[msg.RunID]; !ok {
		return fmt.Errorf("subagent run not found: %s", msg.RunID)
	}
	m.nextMsg++
	msg.ID = m.nextMsg
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now().UTC()
	}
	m.msgs[msg.RunID] = append(m.msgs[msg.RunID], msg)
	return nil
}

func (m *MemoryStore) ListMessages(_ context.Context, runID string) ([]Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := append([]Message(nil), m.msgs[runID]...)
	return out, nil
}

func (m *MemoryStore) AddUsage(_ context.Context, runID string, u llm.Usage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.runs[runID]
	if !ok {
		return fmt.Errorf("subagent run not found: %s", runID)
	}
	r.PromptTokensTotal += int64(u.PromptTokens)
	r.CompletionTokensTotal += int64(u.CompletionTokens)
	r.PromptCacheHitTokensTotal += int64(u.PromptCacheHitTokens)
	m.runs[runID] = r
	return nil
}

func (m *MemoryStore) SumUsage(_ context.Context, parentSessionID string) (llm.Usage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var u llm.Usage
	for _, r := range m.runs {
		if r.ParentSessionID != parentSessionID {
			continue
		}
		u.PromptTokens += int(r.PromptTokensTotal)
		u.CompletionTokens += int(r.CompletionTokensTotal)
		u.PromptCacheHitTokens += int(r.PromptCacheHitTokensTotal)
	}
	return u, nil
}

func (m *MemoryStore) SumEstimatedCostCNY(_ context.Context, parentSessionID string) (float64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var sum float64
	for _, r := range m.runs {
		if r.ParentSessionID != parentSessionID {
			continue
		}
		sum += r.EstimatedCostCNY
	}
	return sum, nil
}
