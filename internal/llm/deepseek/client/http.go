package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/hejunqiu/ds-code/internal/logging"
	"go.uber.org/zap"
)

func (c *Client) doWithRetry(ctx context.Context, body []byte) (*http.Response, error) {
	var lastErr error
	backoff := 500 * time.Millisecond
	for attempt := 0; attempt < 3; attempt++ {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
		resp, err := c.http.Do(httpReq)
		if err != nil {
			lastErr = err
			logging.L().Debug("LLM HTTP attempt failed",
				zap.Int("attempt", attempt+1),
				zap.Int64("backoff_ms", backoff.Milliseconds()),
				zap.String("err_type", "network"),
				zap.Error(err),
			)
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			if !sleepCtx(ctx, backoff) {
				return nil, ctx.Err()
			}
			backoff *= 2
			continue
		}
		if resp.StatusCode == 429 || resp.StatusCode >= 500 {
			lastErr = parseAPIError(resp)
			logging.L().Debug("LLM HTTP attempt retryable",
				zap.Int("attempt", attempt+1),
				zap.Int("status", resp.StatusCode),
				zap.Int64("backoff_ms", backoff.Milliseconds()),
				zap.Error(lastErr),
			)
			resp.Body.Close()
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			if !sleepCtx(ctx, backoff) {
				return nil, ctx.Err()
			}
			backoff *= 2
			continue
		}
		logging.L().Debug("LLM HTTP ok",
			zap.Int("attempt", attempt+1),
			zap.Int("status", resp.StatusCode),
		)
		return resp, nil
	}
	return nil, lastErr
}

func sleepCtx(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

func parseAPIError(resp *http.Response) error {
	b, _ := io.ReadAll(resp.Body)
	var ae apiError
	_ = json.Unmarshal(b, &ae)
	msg := ae.Error.Message
	if msg == "" {
		msg = string(b)
	}
	return fmt.Errorf("deepseek api %d: %s", resp.StatusCode, msg)
}

type apiError struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}
