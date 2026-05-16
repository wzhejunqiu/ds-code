package deepseek

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/llm"
)

// Client implements llm.Client for DeepSeek's OpenAI-compatible API.
type Client struct {
	apiKey  string
	baseURL string
	http    *http.Client
}

// NewClient builds a DeepSeek client from config (API key must be set).
func NewClient(cfg *config.Config) *Client {
	base := strings.TrimSuffix(cfg.LLM.BaseURL, "/")
	if cfg.LLM.StrictTools && !strings.HasSuffix(base, "/beta") {
		base = base + "/beta"
	}
	return &Client{
		apiKey:  cfg.APIKey,
		baseURL: base,
		http: &http.Client{
			Timeout: cfg.LLM.Timeout,
		},
	}
}

type chatRequest struct {
	Model           string           `json:"model"`
	Messages        []map[string]any `json:"messages"`
	Tools           []map[string]any `json:"tools,omitempty"`
	ToolChoice      string           `json:"tool_choice,omitempty"`
	Stream          bool             `json:"stream"`
	StreamOptions   *streamOptions   `json:"stream_options,omitempty"`
	MaxTokens       int              `json:"max_tokens"`
	Thinking        *thinking        `json:"thinking,omitempty"`
	ReasoningEffort string           `json:"reasoning_effort,omitempty"`
	User            string           `json:"user_id,omitempty"`
}

type streamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

type thinking struct {
	Type string `json:"type"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Role             string `json:"role"`
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content"`
			ToolCalls        []struct {
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage usagePayload `json:"usage"`
}

type usagePayload struct {
	PromptTokens             int `json:"prompt_tokens"`
	CompletionTokens         int `json:"completion_tokens"`
	PromptCacheHitTokens     int `json:"prompt_cache_hit_tokens"`
}

type streamChunk struct {
	Choices []struct {
		Delta struct {
			Role             string `json:"role"`
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content"`
			ToolCalls        []struct {
				Index    int    `json:"index"`
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage usagePayload `json:"usage"`
}

type apiError struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

// Chat performs a chat completion (streaming aggregated).
func (c *Client) Chat(ctx context.Context, req llm.Request) (*llm.Response, error) {
	if req.MaxTokens <= 0 {
		req.MaxTokens = 16384
	}
	if req.MaxTokens > MaxOutputTokens {
		req.MaxTokens = MaxOutputTokens
	}

	merged := req.MergedSystem
	if merged == "" {
		merged = MergeSystem("", "", "", "", "")
	}

	body := chatRequest{
		Model:     req.Model,
		Messages:  ToAPIMessages(merged, req.Messages),
		Stream:    req.Stream,
		MaxTokens: req.MaxTokens,
		ToolChoice: "auto",
	}
	if req.Stream {
		body.StreamOptions = &streamOptions{IncludeUsage: true}
	}
	if len(req.Tools) > 0 {
		body.Tools = ToolsToAPI(req.Tools, req.StrictTools)
	}
	if req.ThinkingType != "" {
		body.Thinking = &thinking{Type: req.ThinkingType}
	}
	if req.ReasoningEffort != "" {
		body.ReasoningEffort = req.ReasoningEffort
	}
	if req.UserID != "" {
		body.User = req.UserID
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	httpResp, err := c.doWithRetry(ctx, raw)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode >= 400 {
		return nil, parseAPIError(httpResp)
	}

	if req.Stream {
		return parseStream(httpResp.Body)
	}
	return parseNonStream(httpResp.Body)
}

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

func parseNonStream(r io.Reader) (*llm.Response, error) {
	var cr chatResponse
	if err := json.NewDecoder(r).Decode(&cr); err != nil {
		return nil, err
	}
	return responseFromChat(cr)
}

func parseStream(r io.Reader) (*llm.Response, error) {
	var content, reasoning strings.Builder
	toolAcc := map[int]*llm.ToolCall{}
	finish := ""
	var usage llm.Usage

	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}
		var chunk streamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if chunk.Usage.PromptTokens > 0 || chunk.Usage.CompletionTokens > 0 {
			usage = llm.Usage{
				PromptTokens:         chunk.Usage.PromptTokens,
				CompletionTokens:     chunk.Usage.CompletionTokens,
				PromptCacheHitTokens: chunk.Usage.PromptCacheHitTokens,
			}
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		ch := chunk.Choices[0]
		if ch.FinishReason != "" {
			finish = ch.FinishReason
		}
		if ch.Delta.Content != "" {
			content.WriteString(ch.Delta.Content)
		}
		if ch.Delta.ReasoningContent != "" {
			reasoning.WriteString(ch.Delta.ReasoningContent)
		}
		for _, td := range ch.Delta.ToolCalls {
			tc, ok := toolAcc[td.Index]
			if !ok {
				tc = &llm.ToolCall{}
				toolAcc[td.Index] = tc
			}
			if td.ID != "" {
				tc.ID = td.ID
			}
			if td.Function.Name != "" {
				tc.Name = td.Function.Name
			}
			tc.Arguments += td.Function.Arguments
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}

	calls := make([]llm.ToolCall, 0, len(toolAcc))
	for i := 0; i < len(toolAcc); i++ {
		if tc, ok := toolAcc[i]; ok {
			calls = append(calls, *tc)
		}
	}

	return &llm.Response{
		Content:          content.String(),
		ReasoningContent: reasoning.String(),
		ToolCalls:        calls,
		FinishReason:     finish,
		Usage:            usage,
	}, nil
}

func responseFromChat(cr chatResponse) (*llm.Response, error) {
	if len(cr.Choices) == 0 {
		return nil, fmt.Errorf("deepseek: empty choices")
	}
	ch := cr.Choices[0]
	var calls []llm.ToolCall
	for _, tc := range ch.Message.ToolCalls {
		calls = append(calls, llm.ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		})
	}
	return &llm.Response{
		Content:          ch.Message.Content,
		ReasoningContent: ch.Message.ReasoningContent,
		ToolCalls:        calls,
		FinishReason:     ch.FinishReason,
		Usage: llm.Usage{
			PromptTokens:         cr.Usage.PromptTokens,
			CompletionTokens:     cr.Usage.CompletionTokens,
			PromptCacheHitTokens: cr.Usage.PromptCacheHitTokens,
		},
	}, nil
}

// IsContextTooLong reports whether err indicates context length exceeded.
func IsContextTooLong(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "context") && (strings.Contains(s, "length") || strings.Contains(s, "long"))
}
