package agent

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/logging"
	"go.uber.org/zap"
)

const (
	maxOutputTokensEscalate = 65536
	maxRecoveryAttempts     = 3
	maxRateLimitRetries     = 3
	maxServerErrorRetries   = 2
	continueRecoveryMsg     = "[Continue from where you left off. Do not repeat what you already said.]"
)

// chatWithRecovery wraps LLM.Chat with a multi-strategy recovery loop.
func (r *Runner) chatWithRecovery(ctx context.Context, sessionID string, req llm.Request, state *LoopState) (*llm.Response, error) {
	attempt := 0
	currentReq := req

	for {
		resp, err := r.LLM.Chat(ctx, currentReq)
		if err == nil {
			switch {
			case isLengthFinishReason(resp.FinishReason):
				err = fmt.Errorf("max_tokens: finish_reason=%s", resp.FinishReason)
			case isEmptyTerminalResponse(resp):
				err = fmt.Errorf("empty response")
			default:
				return resp, nil
			}
		}

		logging.L().Warn("LLM error in recovery loop",
			zap.String("session_id", sessionID),
			zap.Int("attempt", attempt),
			zap.Error(err),
		)

		switch {
		case llm.IsContextTooLong(err):
			if !state.CompactRetried {
				state.CompactRetried = true
				state.Transition = TransCompactRetry
				logging.L().Info("context too long → compact retry", zap.String("session_id", sessionID))
				if compactErr := r.Context.CompactAPIContext(ctx, sessionID); compactErr != nil {
					logging.L().Debug("compact failed", zap.Error(compactErr))
				}
				view, maxTokens, prepErr := r.Context.PrepareRequest(ctx, sessionID)
				if prepErr != nil {
					return nil, prepErr
				}
				currentReq.Messages = view.Messages
				currentReq.MergedSystem = view.MergedSystem()
				currentReq.MaxTokens = maxTokens
				attempt++
				continue
			}
			if !state.SnipRetried {
				state.SnipRetried = true
				state.Transition = TransSnipRetry
				logging.L().Info("compact insufficient → snip retry", zap.String("session_id", sessionID))
				r.Context.ForceAggressiveSnip = true
				view, maxTokens, prepErr := r.Context.PrepareRequest(ctx, sessionID)
				r.Context.ForceAggressiveSnip = false
				if prepErr != nil {
					return nil, prepErr
				}
				currentReq.Messages = view.Messages
				currentReq.MergedSystem = view.MergedSystem()
				currentReq.MaxTokens = maxTokens
				attempt++
				continue
			}
			return nil, fmt.Errorf("context too long after compact+snip: %w", err)

		case isMaxTokensError(err):
			if !state.MaxTokensEscalated {
				state.MaxTokensEscalated = true
				state.Transition = TransMaxTokensEscalate
				currentReq.MaxTokens = maxOutputTokensEscalate
				logging.L().Info("max_tokens → escalate to 64K", zap.String("session_id", sessionID))
				attempt++
				continue
			}
			if recovered, recErr := r.tryOutputRecovery(sessionID, state, &currentReq); recovered {
				attempt++
				continue
			} else if recErr != nil {
				return nil, recErr
			}
			return nil, fmt.Errorf("max output recovery exhausted: %w", err)

		case isEmptyResponseError(err):
			if recovered, recErr := r.tryOutputRecovery(sessionID, state, &currentReq); recovered {
				attempt++
				continue
			} else if recErr != nil {
				return nil, recErr
			}
			return nil, fmt.Errorf("output recovery exhausted: %w", err)

		case llm.IsTransientNetworkError(err):
			if state.NetworkRetryCount < maxRecoveryAttempts {
				state.NetworkRetryCount++
				state.Transition = TransNetworkRetry
				logging.L().Info("transient network → retry",
					zap.String("session_id", sessionID),
					zap.Int("count", state.NetworkRetryCount),
				)
				attempt++
				continue
			}
			return nil, fmt.Errorf("network retries exhausted: %w", err)

		case llm.IsRateLimit(err):
			if attempt < maxRateLimitRetries {
				backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
				logging.L().Info("rate limited → backoff", zap.String("session_id", sessionID), zap.Duration("backoff", backoff))
				select {
				case <-time.After(backoff):
				case <-ctx.Done():
					return nil, ctx.Err()
				}
				state.Transition = TransRateLimitRetry
				attempt++
				continue
			}
			return nil, fmt.Errorf("rate limit retries exhausted: %w", err)

		case isServerError(err):
			if attempt < maxServerErrorRetries {
				backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
				logging.L().Info("server error → backoff", zap.String("session_id", sessionID), zap.Duration("backoff", backoff))
				select {
				case <-time.After(backoff):
				case <-ctx.Done():
					return nil, ctx.Err()
				}
				attempt++
				continue
			}
			if !state.FallbackTried {
				if fb := r.fallbackModel(); fb != "" {
					state.FallbackTried = true
					state.Transition = TransModelFallback
					currentReq.Model = fb
					logging.L().Info("server error → model fallback", zap.String("session_id", sessionID), zap.String("model", fb))
					attempt++
					continue
				}
			}
			return nil, fmt.Errorf("server error retries exhausted: %w", err)
		}

		return nil, err
	}
}

func (r *Runner) tryOutputRecovery(sessionID string, state *LoopState, req *llm.Request) (bool, error) {
	if state.OutputRecoveryCount >= maxRecoveryAttempts {
		return false, nil
	}
	state.OutputRecoveryCount++
	state.Transition = TransOutputRecovery
	appendContinueMessage(req)
	logging.L().Info("output recovery", zap.String("session_id", sessionID), zap.Int("count", state.OutputRecoveryCount))
	return true, nil
}

func appendContinueMessage(req *llm.Request) {
	req.Messages = append(req.Messages, llm.Message{
		Role:    "user",
		Content: continueRecoveryMsg,
	})
}

func (r *Runner) fallbackModel() string {
	if r.ForSubagent {
		if r.Cfg.LLM.Subagent.FallbackModel != "" {
			return r.Cfg.LLM.Subagent.FallbackModel
		}
	}
	return r.Cfg.LLM.FallbackModel
}

func isMaxTokensError(err error) bool {
	return llm.IsFinishReasonMaxTokens(err)
}

func isEmptyResponseError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "empty response")
}

func isServerError(err error) bool {
	return llm.IsServerError(err)
}

func isLengthFinishReason(reason string) bool {
	switch strings.ToLower(reason) {
	case "length", "max_tokens", "max_tokens_reached":
		return true
	default:
		return false
	}
}

func isEmptyTerminalResponse(resp *llm.Response) bool {
	if resp == nil {
		return true
	}
	if len(resp.ToolCalls) > 0 {
		return false
	}
	if resp.Content != "" || resp.ReasoningContent != "" {
		return false
	}
	reason := strings.ToLower(resp.FinishReason)
	return reason == "" || reason == "stop"
}
