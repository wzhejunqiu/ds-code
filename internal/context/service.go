package context

import (
	"context"
	"os"
	"time"

	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/llm"
	"github.com/hejunqiu/ds-code/internal/prompt"
	"github.com/hejunqiu/ds-code/internal/role"
	"github.com/hejunqiu/ds-code/internal/logging"
	"github.com/hejunqiu/ds-code/internal/llm/deepseek"
	"github.com/hejunqiu/ds-code/internal/session"
	"github.com/hejunqiu/ds-code/internal/session/subagentstore"
	"github.com/hejunqiu/ds-code/internal/tool"
	"github.com/hejunqiu/ds-code/internal/versioninfo"
	"go.uber.org/zap"
)

// Service builds API context and prepares requests.
type Service struct {
	Cfg        *config.Config
	Store      session.Store
	Subagent   subagentstore.Store
	Tools      *tool.Registry
	LLM        llm.Client
	AgentsMD   string
	Rules      string
	ActiveSkill string
	SkillsText  string

	AtExpander *AtExpander

	// Cached for compact condition A within one user turn.
	userTurnBreakdown *ContextBreakdown
	userTurnCounted   bool
}

// BeginUserTurn resets per-user-turn breakdown cache (condition A).
func (s *Service) BeginUserTurn() {
	s.userTurnBreakdown = nil
	s.userTurnCounted = false
}

// EndUserTurn clears cached breakdown after a user turn completes.
func (s *Service) EndUserTurn() {
	s.userTurnBreakdown = nil
	s.userTurnCounted = false
}

// CachedBreakdown returns the breakdown cached for the current user turn, if any.
func (s *Service) CachedBreakdown() *ContextBreakdown {
	return s.userTurnBreakdown
}

// ExpandUserText expands @file and @dir/ references in a user message.
func (s *Service) ExpandUserText(text string) (string, error) {
	if s.AtExpander == nil {
		return text, nil
	}
	return s.AtExpander.Expand(text)
}

// RefreshGitSnapshot captures git status/diff and stores it on the session.
func (s *Service) RefreshGitSnapshot(ctx context.Context, sessionID, projectRoot string) (string, error) {
	snap, err := CaptureGitSnapshot(projectRoot, s.Cfg.Context.GitSnapshotMaxChars)
	if err != nil {
		return "", err
	}
	err = s.Store.UpdateSession(ctx, sessionID, func(sess *session.Session) error {
		sess.GitSnapshot = snap
		return nil
	})
	if err != nil {
		return "", err
	}
	return snap, nil
}

// PrepareRequest builds the API view and may compact once (conditions A/B).
func (s *Service) PrepareRequest(ctx context.Context, sessionID string) (*APIContextView, int, error) {
	sess, err := s.Store.Get(ctx, sessionID)
	if err != nil {
		return nil, 0, err
	}

	if s.shouldCompact(ctx, sessionID, sess) {
		logging.L().Info("context compact triggered", zap.String("session_id", sessionID), zap.Int64("prompt_tokens", sess.PromptTokensTotal))
		if err := s.CompactAPIContext(ctx, sessionID); err != nil {
			logging.L().Warn("context compact failed", zap.String("session_id", sessionID), zap.Error(err))
		}
	}

	view, err := s.BuildAPIContext(ctx, sessionID)
	if err != nil {
		return nil, 0, err
	}
	maxTokens := s.Cfg.LLM.MaxTokens
	if maxTokens > deepseek.MaxOutputTokens {
		maxTokens = deepseek.MaxOutputTokens
	}
	toolCount := 0
	if s.Tools != nil {
		toolCount = len(s.Tools.Definitions())
	}
	logging.L().Debug("prepare request",
		zap.String("session_id", sessionID),
		zap.Int("messages", len(view.Messages)),
		zap.Int("merged_system_chars", len(view.MergedSystem())),
		zap.Int("tools", toolCount),
		zap.Int("max_tokens", maxTokens),
		zap.Int64("prompt_tokens_total", sess.PromptTokensTotal),
	)
	return view, maxTokens, nil
}

func (s *Service) shouldCompact(ctx context.Context, sessionID string, sess session.Session) bool {
	threshold := s.compactThreshold()
	total := s.sessionPromptTotal(ctx, sessionID, sess)
	if total >= threshold {
		logging.L().Debug("should compact",
			zap.String("session_id", sessionID),
			zap.String("reason", "prompt_total"),
			zap.Int("threshold", threshold),
			zap.Int("total", total),
		)
		return true
	}
	if !s.userTurnCounted {
		view, err := s.BuildAPIContext(ctx, sessionID)
		if err != nil {
			return false
		}
		bd, err := CountBreakdown(view)
		if err != nil {
			return false
		}
		s.userTurnBreakdown = &bd
		s.userTurnCounted = true
		if bd.Total() >= threshold {
			logging.L().Debug("should compact",
				zap.String("session_id", sessionID),
				zap.String("reason", "breakdown_total"),
				zap.Int("threshold", threshold),
				zap.Int("total", bd.Total()),
			)
			return true
		}
		return false
	}
	if s.userTurnBreakdown != nil && s.userTurnBreakdown.Total() >= threshold {
		logging.L().Debug("should compact",
			zap.String("session_id", sessionID),
			zap.String("reason", "cached_breakdown"),
			zap.Int("threshold", threshold),
			zap.Int("total", s.userTurnBreakdown.Total()),
		)
		return true
	}
	return false
}

// BuildAPIContext constructs the next-request snapshot from session history.
func (s *Service) BuildAPIContext(ctx context.Context, sessionID string) (*APIContextView, error) {
	sess, err := s.Store.Get(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	msgs, err := s.Store.ListMessages(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	window := s.Cfg.Context.WindowTokens
	if window <= 0 {
		window = deepseek.ContextWindowTokens
	}

	toolDefs := s.Tools.Definitions()
	skills := s.SkillsText
	cwd, _ := os.Getwd()
	runtimeEnv := prompt.FormatRuntimeEnv(
		s.Cfg.ProjectRoot, cwd, time.Now(), versioninfo.PlatformForPrompt(),
	)
	view := &APIContextView{
		SystemPrompt: prompt.DefaultSystemBase,
		RuntimeEnv:   runtimeEnv,
		AgentsMD:     s.AgentsMD,
		Rules:        s.Rules,
		Skills:       skills,
		GitSnapshot:  sess.GitSnapshot,
		ToolsJSON:    deepseek.ToolsJSON(toolDefs),
		WindowTokens: window,
	}

	turns := session.SplitUserTurns(msgs)
	keep := s.keepRecentTurns()
	var recent []session.UserTurn
	if len(turns) > keep {
		recent = turns[len(turns)-keep:]
	} else {
		recent = turns
	}

	var apiMsgs []llm.Message
	if sess.CompactSummary != "" {
		apiMsgs = append(apiMsgs, compactSummaryMessage(sess.CompactSummary))
	}
	for _, turn := range recent {
		for _, m := range turn.Messages {
			if m.Role == role.System {
				continue // history-only events (e.g. checkpoint rewind)
			}
			if m.ID <= sess.CompactUpToMessageID && sess.CompactSummary != "" {
				continue
			}
			apiMsgs = append(apiMsgs, messageToLLM(m))
		}
	}
	view.Messages = apiMsgs
	logging.L().Debug("build api context",
		zap.String("session_id", sessionID),
		zap.Int("history_msgs", len(msgs)),
		zap.Int("api_msgs", len(apiMsgs)),
		zap.Int64("compact_watermark", sess.CompactUpToMessageID),
		zap.Bool("has_summary", sess.CompactSummary != ""),
		zap.Int("recent_turns", len(recent)),
		zap.Int("window_tokens", window),
	)
	return view, nil
}

func (s *Service) sessionPromptTotal(ctx context.Context, sessionID string, sess session.Session) int {
	total := int(sess.PromptTokensTotal)
	if s.Subagent == nil {
		return total
	}
	u, err := s.Subagent.SumUsage(ctx, sessionID)
	if err != nil {
		return total
	}
	return total + u.PromptTokens
}
