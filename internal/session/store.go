package session

import (
	"context"

	"github.com/wzhejunqiu/ds-code/internal/llm"
)

// Store persists sessions and messages.
type Store interface {
	CreateSession(model, effort, thinking string, permMode PermissionMode, runMode RunMode) (Session, error)
	Create(ctx context.Context, s Session) error
	Get(ctx context.Context, id string) (Session, error)
	ListMessages(ctx context.Context, sessionID string) ([]Message, error)
	AppendMessage(ctx context.Context, msg Message) error
	AddUsage(ctx context.Context, sessionID string, u llm.Usage) error
	UpdateSession(ctx context.Context, sessionID string, fn func(*Session) error) error
	ListSessions(ctx context.Context, limit int) ([]Summary, error)
}
