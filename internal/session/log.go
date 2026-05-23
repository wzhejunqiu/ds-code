package session

import (
	"github.com/hejunqiu/ds-code/internal/llm"
	"github.com/hejunqiu/ds-code/internal/logging"
	"go.uber.org/zap"
)

// LogAppendDebug logs a message append at debug level (content length only).
func LogAppendDebug(m Message) {
	logging.L().Debug("session append message",
		zap.String("session_id", m.SessionID),
		zap.String("role", string(m.Role)),
		zap.Int("content_chars", len(m.Content)),
		zap.String("tool_call_id", m.ToolCallID),
	)
}

// LogAddUsageDebug logs token usage accumulation at debug level.
func LogAddUsageDebug(sessionID string, u llm.Usage) {
	logging.L().Debug("session add usage",
		zap.String("session_id", sessionID),
		zap.Int("prompt_tokens", u.PromptTokens),
		zap.Int("completion_tokens", u.CompletionTokens),
		zap.Int("prompt_cache_hit_tokens", u.PromptCacheHitTokens),
	)
}
