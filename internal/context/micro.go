package context

import (
	"crypto/sha256"
	"fmt"

	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/role"
)

const microThreshold = 4096

// MicroCompress replaces large tool results with a digest placeholder.
// Unlike Snip (which clips old messages based on round count), Micro targets
// individual tool results exceeding microThreshold chars regardless of position.
func MicroCompress(messages []llm.Message) []llm.Message {
	out := make([]llm.Message, len(messages))
	for i, m := range messages {
		out[i] = m
		if m.Role == role.Tool && len(m.Content) > microThreshold {
			digest := sha256.Sum256([]byte(m.Content))
			out[i].Content = fmt.Sprintf("[Tool result digest: sha256=%x, original=%d chars]",
				digest[:8], len(m.Content))
		}
	}
	return out
}
