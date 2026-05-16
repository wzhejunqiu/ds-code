package session

import (
	"context"

	"github.com/hejunqiu/ds-code/internal/llm"
)

// Store persists sessions and messages.
type Store interface {
	CreateSession(model, effort, thinking, permMode, runMode string) (Session, error)
	Create(ctx context.Context, s Session) error
	Get(ctx context.Context, id string) (Session, error)
	ListMessages(ctx context.Context, sessionID string) ([]Message, error)
	AppendMessage(ctx context.Context, msg Message) error
	AddUsage(ctx context.Context, sessionID string, u llm.Usage) error
}
