package deepseek

import (
	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/llm"
	dsclient "github.com/wzhejunqiu/ds-code/internal/llm/deepseek/client"
)

// Client implements llm.Client for DeepSeek's OpenAI-compatible API.
type Client = dsclient.Client

// NewClient builds a DeepSeek client from config (API key must be set).
func NewClient(cfg *config.Config) *Client {
	return dsclient.NewClient(cfg)
}

// IsContextTooLong reports whether err indicates context length exceeded.
func IsContextTooLong(err error) bool {
	return dsclient.IsContextTooLong(err)
}

// ToolsJSON serializes tool defs for token breakdown.
func ToolsJSON(tools []llm.ToolDef) string {
	return dsclient.ToolsJSON(tools)
}
