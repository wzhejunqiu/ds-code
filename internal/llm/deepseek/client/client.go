package client

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/llm"
	"github.com/hejunqiu/ds-code/internal/llm/deepseek/limits"
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

// Chat performs a chat completion (streaming aggregated).
func (c *Client) Chat(ctx context.Context, req llm.Request) (*llm.Response, error) {
	if req.MaxTokens <= 0 {
		req.MaxTokens = 16384
	}
	if req.MaxTokens > limits.MaxOutputTokens {
		req.MaxTokens = limits.MaxOutputTokens
	}

	merged := req.MergedSystem
	if merged == "" {
		merged = MergeSystem("", "", "", "", "")
	}

	body := chatRequest{
		Model:      req.Model,
		Messages:   ToAPIMessages(merged, req.Messages),
		Stream:     req.Stream,
		MaxTokens:  req.MaxTokens,
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
		return parseStream(httpResp.Body, req.OnStream)
	}
	return parseNonStream(httpResp.Body)
}
