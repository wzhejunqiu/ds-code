package context

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/hejunqiu/ds-code/internal/llm"
	"github.com/hejunqiu/ds-code/internal/role"
	"github.com/hejunqiu/ds-code/internal/llm/deepseek"
	"github.com/hejunqiu/ds-code/internal/logging"
	"github.com/hejunqiu/ds-code/internal/session"
	"go.uber.org/zap"
)

var secretLineRE = regexp.MustCompile(`(?i)(api[_-]?key|secret|password|token)\s*[=:]\s*\S+`)

// CompactAPIContext summarizes older turns via LLM and updates session watermark.
func (s *Service) CompactAPIContext(ctx context.Context, sessionID string) error {
	logging.L().Info("compact start", zap.String("session_id", sessionID))
	sess, err := s.Store.Get(ctx, sessionID)
	if err != nil {
		return err
	}
	msgs, err := s.Store.ListMessages(ctx, sessionID)
	if err != nil {
		return err
	}
	turns := session.SplitUserTurns(msgs)
	keep := s.keepRecentTurns()
	if len(turns) <= keep {
		return nil
	}
	oldTurns := turns[:len(turns)-keep]
	if len(oldTurns) == 0 {
		return nil
	}

	transcript := formatTurnsForCompact(oldTurns)
	transcript = sanitizeCompactInput(transcript)
	logging.L().Debug("compact preparing",
		zap.String("session_id", sessionID),
		zap.Int("old_turns", len(oldTurns)),
		zap.Int("transcript_chars", len(transcript)),
	)
	if strings.TrimSpace(transcript) == "" {
		return nil
	}

	summary, usage, err := s.summarize(ctx, sess, transcript)
	if err != nil {
		_ = s.compactFallback(ctx, sessionID, turns, keep)
		return fmt.Errorf("compact: %w (applied fallback truncation)", err)
	}
	logging.L().Debug("compact summarized",
		zap.String("session_id", sessionID),
		zap.String("model", sess.Model),
		zap.Int("summary_chars", len(summary)),
		zap.Int("prompt_tokens", usage.PromptTokens),
		zap.Int("completion_tokens", usage.CompletionTokens),
	)

	watermark := oldTurns[len(oldTurns)-1].MaxMessageID()
	mergedSummary := sess.CompactSummary
	if mergedSummary != "" {
		mergedSummary = mergedSummary + "\n\n" + summary
	} else {
		mergedSummary = summary
	}

	if err := s.Store.UpdateSession(ctx, sessionID, func(st *session.Session) error {
		st.CompactSummary = mergedSummary
		if watermark > st.CompactUpToMessageID {
			st.CompactUpToMessageID = watermark
		}
		return nil
	}); err != nil {
		return err
	}
	if usage.PromptTokens > 0 || usage.CompletionTokens > 0 {
		_ = s.Store.AddUsage(ctx, sessionID, usage)
	}
	logging.L().Info("compact done",
		zap.String("session_id", sessionID),
		zap.Int64("watermark", watermark),
		zap.Int("summary_chars", len(mergedSummary)),
	)
	return nil
}

func (s *Service) summarize(ctx context.Context, sess session.Session, transcript string) (string, llm.Usage, error) {
	if s.LLM == nil {
		return "", llm.Usage{}, fmt.Errorf("no LLM client configured")
	}
	maxOut := 4096
	if s.Cfg.LLM.MaxTokens > 0 && s.Cfg.LLM.MaxTokens < maxOut {
		maxOut = s.Cfg.LLM.MaxTokens
	}
	prompt := CompactSummarizeUserPrefix + transcript

	resp, err := s.LLM.Chat(ctx, llm.Request{
		MergedSystem: CompactSummarizeSystem,
		Messages:     []llm.Message{{Role: role.User, Content: prompt}},
		Model:        sess.Model,
		MaxTokens:    maxOut,
		Stream:        false,
		ThinkingType:  "disabled",
		StrictTools:  s.Cfg.LLM.StrictTools,
		UserID:       "compact-" + sess.ID,
	})
	if err != nil {
		return "", llm.Usage{}, err
	}
	return strings.TrimSpace(resp.Content), resp.Usage, nil
}

func (s *Service) compactFallback(ctx context.Context, sessionID string, turns []session.UserTurn, keep int) error {
	if len(turns) <= keep {
		return nil
	}
	old := turns[:len(turns)-keep]
	watermark := old[len(old)-1].MaxMessageID()
	logging.L().Debug("compact fallback",
		zap.String("session_id", sessionID),
		zap.Int64("watermark_msg_id", watermark),
		zap.Bool("set_fallback_summary", true),
	)
	return s.Store.UpdateSession(ctx, sessionID, func(st *session.Session) error {
		if watermark > st.CompactUpToMessageID {
			st.CompactUpToMessageID = watermark
			if st.CompactSummary == "" {
				st.CompactSummary = CompactFallbackSummary
			}
		}
		return nil
	})
}

func formatTurnsForCompact(turns []session.UserTurn) string {
	var b strings.Builder
	for i, t := range turns {
		fmt.Fprintf(&b, CompactTurnLabel, i+1)
		for _, m := range t.Messages {
			switch m.Role {
			case role.System:
				continue
			case role.User:
				b.WriteString(CompactRoleUser)
				b.WriteString(truncateCompact(m.Content, 8000))
				b.WriteByte('\n')
			case role.Assistant:
				if m.ReasoningContent != "" {
					b.WriteString(CompactRoleAssistantReason)
					b.WriteString(truncateCompact(m.ReasoningContent, 4000))
					b.WriteByte('\n')
				}
				if m.Content != "" {
					b.WriteString(CompactRoleAssistant)
					b.WriteString(truncateCompact(m.Content, 8000))
					b.WriteByte('\n')
				}
			case role.Tool:
				b.WriteString(CompactRoleTool)
				b.WriteString(m.ToolName)
				b.WriteString(": ")
				b.WriteString(truncateCompact(m.Content, 4000))
				b.WriteByte('\n')
			}
		}
	}
	return b.String()
}

func truncateCompact(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + CompactTruncated
}

func sanitizeCompactInput(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if secretLineRE.MatchString(line) {
			lines[i] = CompactRedacted
		}
	}
	return strings.Join(lines, "\n")
}

func (s *Service) compactThreshold() int {
	ratio := s.Cfg.Context.CompactThresholdRatio
	if ratio <= 0 {
		ratio = 0.80
	}
	window := s.Cfg.Context.WindowTokens
	if window <= 0 {
		window = deepseek.ContextWindowTokens
	}
	return int(float64(window) * ratio)
}

func (s *Service) keepRecentTurns() int {
	n := s.Cfg.Context.KeepRecentTurns
	if n <= 0 {
		return 6
	}
	return n
}
