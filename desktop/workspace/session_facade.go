package workspace

import (
	"context"
	"fmt"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/role"
	"github.com/wzhejunqiu/ds-code/internal/session"
)

// ListChats returns agent conversation windows for a workspace.
func (m *Manager) ListChats(wsID string) ([]ChatSummary, error) {
	rt, err := m.Ensure(wsID)
	if err != nil {
		return nil, err
	}
	list, err := rt.store.ListSessions(context.Background(), 100)
	if err != nil {
		return nil, err
	}
	out := make([]ChatSummary, 0, len(list))
	for _, s := range list {
		out = append(out, chatSummaryFromList(s))
	}
	return out, nil
}

func chatSummaryFromList(s session.Summary) ChatSummary {
	title := s.Title
	if title == "" {
		title = "(untitled)"
	}
	return ChatSummary{
		ID:        s.ID,
		Title:     title,
		Model:     s.Model,
		UpdatedAt: s.UpdatedAt.UnixMilli(),
		CreatedAt: s.CreatedAt.UnixMilli(),
	}
}

// ResumeChat loads session metadata and message history.
func (m *Manager) ResumeChat(wsID, sessionID string) ([]ChatMessage, ChatSummary, error) {
	rt, err := m.Ensure(wsID)
	if err != nil {
		return nil, ChatSummary{}, err
	}
	ctx := context.Background()
	sess, err := rt.store.Get(ctx, sessionID)
	if err != nil {
		return nil, ChatSummary{}, err
	}
	msgs, err := rt.store.ListMessages(ctx, sessionID)
	if err != nil {
		return nil, ChatSummary{}, err
	}
	return messagesToDTO(msgs), chatSummaryFromSession(sess), nil
}

func messagesToDTO(msgs []session.Message) []ChatMessage {
	out := make([]ChatMessage, 0, len(msgs))
	for _, msg := range msgs {
		r := string(msg.Role)
		if msg.Role == role.Tool {
			r = "tool"
		}
		out = append(out, ChatMessage{
			ID:            msg.ID,
			Role:          r,
			Content:       msg.Content,
			ContentFormat: msg.ContentFormat,
			Reasoning:     msg.ReasoningContent,
			ToolCalls:     msg.ToolCallsJSON,
			ToolCallID:    msg.ToolCallID,
			ToolName:      msg.ToolName,
			CreatedAt:     msg.CreatedAt.UnixMilli(),
		})
	}
	return out
}

// RenameChat updates the session title.
func (m *Manager) RenameChat(wsID, sessionID, title string) error {
	rt, err := m.Ensure(wsID)
	if err != nil {
		return err
	}
	return rt.store.UpdateSession(context.Background(), sessionID, func(s *session.Session) error {
		s.Title = title
		s.UpdatedAt = time.Now().UTC()
		return nil
	})
}

// DeleteChat hides a session by clearing its title and marking updated (messages remain).
func (m *Manager) DeleteChat(wsID, sessionID string) error {
	rt, err := m.Ensure(wsID)
	if err != nil {
		return err
	}
	// Sessions with messages still appear in ListSessions; we rename to mark hidden.
	hiddenTitle := fmt.Sprintf("[deleted] %s", sessionID[:8])
	return rt.store.UpdateSession(context.Background(), sessionID, func(s *session.Session) error {
		s.Title = hiddenTitle
		s.UpdatedAt = time.Now().UTC()
		return nil
	})
}

// ProjectRoot returns the project root for a workspace.
func (m *Manager) ProjectRoot(wsID string) (string, error) {
	for _, e := range m.registry.Data().Workspaces {
		if e.ID == wsID {
			return e.Root, nil
		}
	}
	return "", fmt.Errorf("workspace not found: %s", wsID)
}
