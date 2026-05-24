package session

import (
	"context"

	"github.com/wzhejunqiu/ds-code/internal/role"
)

// IsFirstUserMessage reports whether the session has exactly one user message.
func IsFirstUserMessage(ctx context.Context, store Store, sessionID string) (bool, error) {
	msgs, err := store.ListMessages(ctx, sessionID)
	if err != nil {
		return false, err
	}
	n := 0
	for _, m := range msgs {
		if m.Role == role.User {
			n++
		}
	}
	return n == 1, nil
}
