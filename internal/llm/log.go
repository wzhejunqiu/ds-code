package llm

import (
	"github.com/wzhejunqiu/ds-code/internal/logging"
	"go.uber.org/zap"
)

// LogResponseDebug logs an aggregated chat completion response at debug level.
func LogResponseDebug(resp *Response) {
	if resp == nil {
		return
	}
	logging.L().Debug("LLM response",
		zap.String("finish_reason", resp.FinishReason),
		zap.Int("content_chars", len(resp.Content)),
		zap.Int("reasoning_chars", len(resp.ReasoningContent)),
		zap.Int("tool_calls", len(resp.ToolCalls)),
		zap.Int("prompt_tokens", resp.Usage.PromptTokens),
		zap.Int("completion_tokens", resp.Usage.CompletionTokens),
		zap.Int("prompt_cache_hit_tokens", resp.Usage.PromptCacheHitTokens),
	)
}

// LogRequestDebug logs the outgoing chat completion request at debug level.
func LogRequestDebug(req Request, apiBody []byte) {
	logging.L().Debug("LLM request",
		zap.String("model", req.Model),
		zap.Int("max_tokens", req.MaxTokens),
		zap.Bool("stream", req.Stream),
		zap.String("thinking_type", req.ThinkingType),
		zap.String("reasoning_effort", req.ReasoningEffort),
		zap.String("user_id", req.UserID),
		zap.Bool("strict_tools", req.StrictTools),
		zap.Int("messages", len(req.Messages)),
		zap.Int("tools", len(req.Tools)),
		logging.FieldBody(apiBody),
	)
}
