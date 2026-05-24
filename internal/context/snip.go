package context

import (
	"fmt"

	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/role"
)

const snipPlaceholderFmt = "[Tool result snipped: %d chars]"

// SnipToolResults replaces old tool result content with placeholder text
// to reduce token usage. Only tool messages with role=tool are snipped.
// Messages within the last keepRounds full assistant→tool cycles are preserved intact.
func SnipToolResults(messages []llm.Message, keepRounds int) []llm.Message {
	if keepRounds <= 0 {
		keepRounds = 3
	}

	// Count assistant messages from the end to find the cutoff.
	assistantCount := 0
	cutoff := len(messages)
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == role.Assistant {
			assistantCount++
			if assistantCount > keepRounds {
				cutoff = i
				break
			}
		}
	}

	out := make([]llm.Message, len(messages))
	for i, m := range messages {
		out[i] = m
		if i < cutoff && m.Role == role.Tool && len(m.Content) > 200 {
			out[i].Content = fmt.Sprintf(snipPlaceholderFmt, len(m.Content))
		}
	}
	return out
}
