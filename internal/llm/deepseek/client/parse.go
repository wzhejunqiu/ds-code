package client

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/logging"
	"go.uber.org/zap"
)

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
	PromptTokens         int `json:"prompt_tokens"`
	CompletionTokens     int `json:"completion_tokens"`
	PromptCacheHitTokens int `json:"prompt_cache_hit_tokens"`
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

func parseNonStream(r io.Reader) (*llm.Response, error) {
	var cr chatResponse
	if err := json.NewDecoder(r).Decode(&cr); err != nil {
		return nil, err
	}
	return responseFromChat(cr)
}

func parseStream(r io.Reader, onStream llm.StreamFunc) (*llm.Response, error) {
	var content, reasoning strings.Builder
	toolAcc := map[int]*llm.ToolCall{}
	finish := ""
	var usage llm.Usage
	var sseLines, jsonErrors int

	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		sseLines++
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}
		var chunk streamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			jsonErrors++
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
			if onStream != nil {
				onStream(llm.StreamDelta{Content: ch.Delta.Content})
			}
		}
		if ch.Delta.ReasoningContent != "" {
			reasoning.WriteString(ch.Delta.ReasoningContent)
			if onStream != nil {
				onStream(llm.StreamDelta{Reasoning: ch.Delta.ReasoningContent})
			}
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

	logging.L().Debug("LLM stream parsed",
		zap.Int("sse_lines", sseLines),
		zap.Int("json_errors", jsonErrors),
		zap.String("finish_reason", finish),
		zap.Int("content_chars", content.Len()),
		zap.Int("reasoning_chars", reasoning.Len()),
		zap.Int("tool_calls", len(calls)),
		zap.Int("prompt_tokens", usage.PromptTokens),
		zap.Int("completion_tokens", usage.CompletionTokens),
	)

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
