package context

import (
	"encoding/json"
	"strings"
	"unicode/utf8"

	"github.com/hejunqiu/ds-code/internal/llm"
	"github.com/hejunqiu/ds-code/internal/llm/deepseek"
	tokdeepseek "github.com/hejunqiu/ds-code/internal/tokenizer/deepseek"
)

// ContextBreakdown splits the next request into six display buckets.
type ContextBreakdown struct {
	SystemPrompt int
	Tools        int
	Rules        int
	Skills       int
	Subagents    int
	Conversation int
	Window       int
	Estimated    bool
}

// Total returns tokens/chars for compact condition A (excludes Rules/Skills display-only).
func (b ContextBreakdown) Total() int {
	return b.SystemPrompt + b.Tools + b.Subagents + b.Conversation
}

// PercentOfWindow returns part as a fraction of the context window.
func (b ContextBreakdown) PercentOfWindow(part int) float64 {
	if b.Window <= 0 {
		return 0
	}
	return float64(part) / float64(b.Window) * 100
}

// PercentOfTotal returns part as a fraction of Total().
func (b ContextBreakdown) PercentOfTotal(part int) float64 {
	t := b.Total()
	if t <= 0 {
		return 0
	}
	return float64(part) / float64(t) * 100
}

// Counter counts text units (tokens or char estimate).
type Counter interface {
	Count(text string) int
}

type charCounter struct{}

func (charCounter) Count(text string) int {
	n := utf8.RuneCountInString(text)
	if n == 0 {
		return 0
	}
	return (n + 3) / 4
}

type tokenizerCounter struct {
	tok *tokdeepseek.Tokenizer
}

func (c tokenizerCounter) Count(text string) int {
	return c.tok.Count(text)
}

func defaultCounter() (Counter, bool) {
	tok, err := tokdeepseek.Default()
	if err != nil {
		return charCounter{}, true
	}
	return tokenizerCounter{tok: tok}, false
}

// CountBreakdown estimates the next request size from an APIContextView.
func CountBreakdown(view *APIContextView) (ContextBreakdown, error) {
	counter, estimated := defaultCounter()
	window := deepseek.ContextWindowTokens
	if view != nil && view.WindowTokens > 0 {
		window = view.WindowTokens
	}

	merged := view.MergedSystem()
	bd := ContextBreakdown{
		SystemPrompt: counter.Count(merged),
		Tools:        counter.Count(view.ToolsJSON),
		Rules:        counter.Count(strings.TrimSpace(view.Rules)),
		Skills:       counter.Count(strings.TrimSpace(view.Skills)),
		Window:       window,
		Estimated:    estimated,
	}

	var subagentParts, convParts []string
	for _, m := range view.Messages {
		s := serializeMessageForCount(m)
		if m.Role == "tool" && m.Name == "task" {
			subagentParts = append(subagentParts, s)
		} else {
			convParts = append(convParts, s)
		}
	}
	bd.Subagents = counter.Count(strings.Join(subagentParts, ""))
	bd.Conversation = counter.Count(strings.Join(convParts, ""))
	return bd, nil
}

func serializeMessageForCount(m llm.Message) string {
	switch m.Role {
	case "assistant":
		var b strings.Builder
		if m.ReasoningContent != "" {
			b.WriteString(m.ReasoningContent)
		}
		if m.Content != "" {
			b.WriteString(m.Content)
		}
		if len(m.ToolCalls) > 0 {
			raw, _ := json.Marshal(m.ToolCalls)
			b.Write(raw)
		}
		return b.String()
	case "tool":
		return m.Content
	default:
		return m.Content
	}
}
