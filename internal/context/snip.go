package context

import (
	"fmt"

	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/role"
)

const (
	snipPlaceholderFmt = "[Tool result snipped: %d chars]"
	snipMinChars       = 200
)

// SnipToolResults replaces old tool result content with placeholder text
// to reduce token usage. Only tool messages with role=tool are snipped.
// The current user turn (messages after the last user message) is never snipped.
// When keepRounds > 0, the last keepRounds user turns are preserved intact.
func SnipToolResults(messages []llm.Message, keepRounds int) []llm.Message {
	protectedFrom := snipProtectedFrom(messages, keepRounds)

	out := make([]llm.Message, len(messages))
	for i, m := range messages {
		out[i] = m
		if i >= protectedFrom {
			continue
		}
		if m.Role == role.Tool && len(m.Content) > snipMinChars {
			out[i].Content = fmt.Sprintf(snipPlaceholderFmt, len(m.Content))
		}
	}
	return out
}

func lastUserIndex(messages []llm.Message) int {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == role.User {
			return i
		}
	}
	return -1
}

// snipProtectedFrom returns the first index that must not be snipped.
// keepRounds <= 0 (aggressive recovery): protect the current user turn only.
// keepRounds > 0: protect the last keepRounds user turns.
func snipProtectedFrom(messages []llm.Message, keepRounds int) int {
	lastUser := lastUserIndex(messages)
	if lastUser < 0 {
		return len(messages)
	}
	if keepRounds <= 0 {
		return lastUser + 1
	}
	userCount := 0
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == role.User {
			userCount++
			if userCount >= keepRounds {
				return i
			}
		}
	}
	return 0
}
