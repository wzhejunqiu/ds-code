package web_fetch

import (
	"context"
	"fmt"
	"strings"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/datadir"
	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/logging"
	"github.com/wzhejunqiu/ds-code/internal/role"
	"go.uber.org/zap"
)

const (
	analyzeSystem        = "You analyze web page content provided in Markdown. Answer the user's question using only information from the page. If the page does not contain the answer, say so clearly."
	analyzeUserSep       = "\n\n---\n\nPage content:\n\n"
	maxAnalyzeInputBytes = 96 * 1024
)

// AnalyzePage sends page markdown and the user prompt to the fetch model.
func AnalyzePage(ctx context.Context, client llm.Client, cfg *config.Config, prompt, markdown string) (string, error) {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return "", fmt.Errorf("%s", ErrPromptRequired)
	}
	markdown = truncateUTF8(markdown, maxAnalyzeInputBytes)
	userContent := prompt + analyzeUserSep + markdown

	maxOut := 4096
	if cfg.LLM.MaxTokens > 0 && cfg.LLM.MaxTokens < maxOut {
		maxOut = cfg.LLM.MaxTokens
	}

	resp, err := client.Chat(ctx, llm.Request{
		MergedSystem: analyzeSystem,
		Messages:     []llm.Message{{Role: role.User, Content: userContent}},
		Model:        cfg.Web.ResolveFetchModel(),
		MaxTokens:    maxOut,
		Stream:       false,
		ThinkingType: "disabled",
		StrictTools:  cfg.LLM.StrictTools,
		UserID:       datadir.Identifier(),
	})
	if err != nil {
		return "", err
	}
	logging.L().Debug("web_fetch analyze",
		zap.Int("prompt_tokens", resp.Usage.PromptTokens),
		zap.Int("completion_tokens", resp.Usage.CompletionTokens),
	)
	return strings.TrimSpace(resp.Content), nil
}

func truncateUTF8(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	end := maxBytes
	for end > 0 && s[end]&0xC0 == 0x80 {
		end--
	}
	return s[:end] + "\n...（内容已截断）"
}
