package context

import (
	"context"

	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/llm"
	"github.com/hejunqiu/ds-code/internal/role"
	"github.com/hejunqiu/ds-code/internal/logging"
	"github.com/hejunqiu/ds-code/internal/llm/deepseek"
	"github.com/hejunqiu/ds-code/internal/session"
	"github.com/hejunqiu/ds-code/internal/tool"
	"go.uber.org/zap"
)

// Service builds API context and prepares requests.
type Service struct {
	Cfg        *config.Config
	Store      session.Store
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
	return view, maxTokens, nil
}

func (s *Service) shouldCompact(ctx context.Context, sessionID string, sess session.Session) bool {
	threshold := s.compactThreshold()
	if int(sess.PromptTokensTotal) >= threshold {
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
		return bd.Total() >= threshold
	}
	if s.userTurnBreakdown != nil {
		return s.userTurnBreakdown.Total() >= threshold
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
	view := &APIContextView{
		SystemPrompt: defaultSystemBase,
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
	return view, nil
}
