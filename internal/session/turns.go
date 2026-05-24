package session

import "github.com/wzhejunqiu/ds-code/internal/role"

// UserTurn is one user message and all following assistant/tool messages until the next user.
type UserTurn struct {
	Messages []Message
}

// SplitUserTurns groups messages by user turn boundaries.
func SplitUserTurns(msgs []Message) []UserTurn {
	var turns []UserTurn
	var current *UserTurn
	for _, m := range msgs {
		if m.Role == role.User {
			if current != nil {
				turns = append(turns, *current)
			}
			current = &UserTurn{Messages: []Message{m}}
			continue
		}
		if current == nil {
			continue
		}
		current.Messages = append(current.Messages, m)
	}
	if current != nil {
		turns = append(turns, *current)
	}
	return turns
}

// MaxMessageID returns the highest message ID in the turn, or 0.
func (t UserTurn) MaxMessageID() int64 {
	var max int64
	for _, m := range t.Messages {
		if m.ID > max {
			max = m.ID
		}
	}
	return max
}

// FirstUserContent returns the first user message content in the turn.
func (t UserTurn) FirstUserContent() string {
	for _, m := range t.Messages {
		if m.Role == role.User {
			return m.Content
		}
	}
	return ""
}
