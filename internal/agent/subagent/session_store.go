package subagent

import (
	"context"
	"fmt"

	"github.com/hejunqiu/ds-code/internal/llm"
	"github.com/hejunqiu/ds-code/internal/session"
	"github.com/hejunqiu/ds-code/internal/session/subagentstore"
)

// sessionStore adapts subagentstore.Store to session.Store for one subagent run.
type sessionStore struct {
	sub   subagentstore.Store
	runID string
	run   subagentstore.Run
}

func newSessionStore(sub subagentstore.Store, run subagentstore.Run) *sessionStore {
	return &sessionStore{sub: sub, runID: run.ID, run: run}
}

func (s *sessionStore) CreateSession(model, effort, thinking, permMode, runMode string) (session.Session, error) {
	return s.toSession(), nil
}

func (s *sessionStore) NewSession(model, effort, thinking, permMode, runMode string) (session.Session, error) {
	return s.CreateSession(model, effort, thinking, permMode, runMode)
}

func (s *sessionStore) Create(_ context.Context, sess session.Session) error {
	return fmt.Errorf("subagent session store: create not supported")
}

func (s *sessionStore) Get(ctx context.Context, _ string) (session.Session, error) {
	r, err := s.sub.GetRun(ctx, s.runID)
	if err != nil {
		return session.Session{}, err
	}
	s.run = r
	return s.toSession(), nil
}

func (s *sessionStore) ListMessages(ctx context.Context, _ string) ([]session.Message, error) {
	msgs, err := s.sub.ListMessages(ctx, s.runID)
	if err != nil {
		return nil, err
	}
	out := make([]session.Message, len(msgs))
	for i, m := range msgs {
		out[i] = subagentMessageToSession(m)
	}
	return out, nil
}

func (s *sessionStore) AppendMessage(ctx context.Context, msg session.Message) error {
	return s.sub.AppendMessage(ctx, sessionMessageToSubagent(msg, s.runID))
}

func (s *sessionStore) AddUsage(ctx context.Context, _ string, u llm.Usage) error {
	return s.sub.AddUsage(ctx, s.runID, u)
}

func (s *sessionStore) UpdateSession(ctx context.Context, _ string, fn func(*session.Session) error) error {
	sess := s.toSession()
	if err := fn(&sess); err != nil {
		return err
	}
	s.run.PromptTokensTotal = sess.PromptTokensTotal
	s.run.CompletionTokensTotal = sess.CompletionTokensTotal
	s.run.PromptCacheHitTokensTotal = sess.PromptCacheHitTokensTotal
	return nil
}

func (s *sessionStore) ListSessions(_ context.Context, _ int) ([]session.Summary, error) {
	return nil, fmt.Errorf("subagent session store: list sessions not supported")
}

func (s *sessionStore) toSession() session.Session {
	return session.Session{
		ID:                        s.runID,
		Model:                     s.run.Model,
		ReasoningEffort:           s.run.ReasoningEffort,
		ThinkingType:              s.run.ThinkingType,
		PermissionMode:            "readonly",
		RunMode:                   "agent",
		PromptTokensTotal:         s.run.PromptTokensTotal,
		CompletionTokensTotal:     s.run.CompletionTokensTotal,
		PromptCacheHitTokensTotal: s.run.PromptCacheHitTokensTotal,
		CreatedAt:                 s.run.CreatedAt,
		UpdatedAt:                 s.run.CreatedAt,
	}
}

func subagentMessageToSession(m subagentstore.Message) session.Message {
	return session.Message{
		ID:                   m.ID,
		SessionID:            m.RunID,
		Role:                 m.Role,
		Content:              m.Content,
		ReasoningContent:     m.ReasoningContent,
		ReasoningDurationMS:  m.ReasoningDurationMS,
		TurnDurationMS:       m.TurnDurationMS,
		ToolCallsJSON:        m.ToolCallsJSON,
		ToolCallID:           m.ToolCallID,
		ToolName:             m.ToolName,
		PromptTokens:         m.PromptTokens,
		CompletionTokens:     m.CompletionTokens,
		PromptCacheHitTokens: m.PromptCacheHitTokens,
		CreatedAt:            m.CreatedAt,
	}
}

func sessionMessageToSubagent(m session.Message, runID string) subagentstore.Message {
	return subagentstore.Message{
		ID:                   m.ID,
		RunID:                runID,
		Role:                 m.Role,
		Content:              m.Content,
		ReasoningContent:     m.ReasoningContent,
		ReasoningDurationMS:  m.ReasoningDurationMS,
		TurnDurationMS:       m.TurnDurationMS,
		ToolCallsJSON:        m.ToolCallsJSON,
		ToolCallID:           m.ToolCallID,
		ToolName:             m.ToolName,
		PromptTokens:         m.PromptTokens,
		CompletionTokens:     m.CompletionTokens,
		PromptCacheHitTokens: m.PromptCacheHitTokens,
		CreatedAt:            m.CreatedAt,
	}
}
