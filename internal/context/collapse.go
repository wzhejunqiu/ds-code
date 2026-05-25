package context

import (
	"context"
	"fmt"
	"strings"

	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/llm/deepseek"
	"github.com/wzhejunqiu/ds-code/internal/logging"
	"github.com/wzhejunqiu/ds-code/internal/role"
	"go.uber.org/zap"
)

const maxCollapsePasses = 3

// CollapseState tracks in-memory collapse for one session (view layer only).
type CollapseState struct {
	CollapsedTurns int
	Summary        string
}

type collapseTracker struct {
	bySess map[string]*CollapseState
}

func newCollapseTracker() *collapseTracker {
	return &collapseTracker{bySess: make(map[string]*CollapseState)}
}

func (c *collapseTracker) get(sessionID string) *CollapseState {
	st, ok := c.bySess[sessionID]
	if !ok {
		st = &CollapseState{}
		c.bySess[sessionID] = st
	}
	return st
}

func (s *Service) collapseThreshold() int {
	ratio := s.Cfg.Context.CollapseThresholdRatio
	if ratio <= 0 {
		ratio = 0.85
	}
	window := s.Cfg.Context.WindowTokens
	if window <= 0 {
		window = deepseek.ContextWindowTokens
	}
	return int(float64(window) * ratio)
}

// applyCollapseIfNeeded performs L3 view-layer collapse when over threshold.
func (s *Service) applyCollapseIfNeeded(ctx context.Context, sessionID string, view *APIContextView) {
	if view == nil || len(view.Messages) == 0 {
		return
	}
	if s.collapse == nil {
		s.collapse = newCollapseTracker()
	}
	st := s.collapse.get(sessionID)
	keep := s.keepRecentTurns()
	for pass := 0; pass < maxCollapsePasses; pass++ {
		bd, err := CountBreakdown(view)
		if err != nil || bd.Total() < s.collapseThreshold() {
			return
		}
		turns := splitLLMTurns(view.Messages)
		if len(turns) <= keep+1 {
			return
		}
		collapseN := 1
		if st.CollapsedTurns > 0 {
			// Already collapsed: fold one more oldest turn from the remaining window.
			if len(turns) <= keep+1 {
				return
			}
		} else {
			collapseN = len(turns) - keep
			if collapseN < 1 {
				return
			}
		}
		transcript := formatLLMTurnsFromMessages(turns[:collapseN], view.Messages)
		summary := collapseFallbackSummary(transcript, 2000)
		if s.LLM != nil {
			sess, err := s.Store.Get(ctx, sessionID)
			if err == nil {
				if sum, _, err := s.summarize(ctx, sess, sanitizeCompactInput(transcript)); err == nil && sum != "" {
					summary = sum
				}
			}
		}
		if st.Summary != "" {
			summary = st.Summary + "\n\n" + summary
		}
		if len(summary) > 8000 {
			summary = summary[:8000] + "\n…(truncated)"
		}
		st.Summary = summary
		st.CollapsedTurns += collapseN
		view.Messages = buildCollapsedMessages(view.Messages, collapseN, st.CollapsedTurns, summary)
		logging.L().Info("context collapse applied",
			zap.String("session_id", sessionID),
			zap.Int("collapsed_turns", collapseN),
			zap.Int("total_collapsed", st.CollapsedTurns),
			zap.Int("pass", pass+1),
		)
	}
}

type llmTurn struct {
	start int
	end   int
}

func isCollapseMarker(m llm.Message) bool {
	if m.Role != role.User {
		return false
	}
	c := m.Content
	return strings.Contains(c, "<conversation-summary>") ||
		strings.HasPrefix(strings.TrimSpace(c), "[collapsed ")
}

func splitLLMTurns(msgs []llm.Message) []llmTurn {
	var turns []llmTurn
	for i, m := range msgs {
		if m.Role != role.User {
			continue
		}
		if isCollapseMarker(m) {
			continue
		}
		if len(turns) > 0 {
			turns[len(turns)-1].end = i
		}
		turns = append(turns, llmTurn{start: i, end: len(msgs)})
	}
	if len(turns) > 0 {
		turns[len(turns)-1].end = len(msgs)
	}
	return turns
}

func formatLLMTurnsFromMessages(turns []llmTurn, msgs []llm.Message) string {
	var b strings.Builder
	for _, t := range turns {
		for i := t.start; i < t.end && i < len(msgs); i++ {
			m := msgs[i]
			b.WriteString(string(m.Role))
			b.WriteString(": ")
			b.WriteString(m.Content)
			b.WriteByte('\n')
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func buildCollapsedMessages(msgs []llm.Message, collapseTurns, totalCollapsed int, summary string) []llm.Message {
	turns := splitLLMTurns(msgs)
	if collapseTurns >= len(turns) {
		collapseTurns = len(turns) - 1
	}
	if collapseTurns < 1 {
		return msgs
	}
	cut := turns[collapseTurns].start
	out := make([]llm.Message, 0, len(msgs)-cut+2)
	out = append(out, llm.Message{
		Role:    role.User,
		Content: fmt.Sprintf("<conversation-summary>\n%s\n</conversation-summary>", summary),
	})
	out = append(out, llm.Message{
		Role:    role.User,
		Content: fmt.Sprintf("[collapsed %d earlier turns]", totalCollapsed),
	})
	out = append(out, msgs[cut:]...)
	return out
}

const collapseFallbackPerLine = 200

func collapseFallbackSummary(transcript string, maxChars int) string {
	if maxChars <= 0 {
		maxChars = 2000
	}
	var b strings.Builder
	for _, line := range strings.Split(transcript, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if len(line) > collapseFallbackPerLine {
			line = line[:collapseFallbackPerLine] + "…"
		}
		if b.Len()+len(line)+1 > maxChars {
			break
		}
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(line)
	}
	out := b.String()
	if out == "" {
		if len(transcript) > maxChars {
			return transcript[:maxChars] + "\n…(collapsed excerpt)"
		}
		return transcript
	}
	if len(transcript) > len(out) {
		out += "\n…(collapsed excerpt)"
	}
	return out
}
