package context

import (
	"context"
	"encoding/json"

	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/llm"
	"github.com/hejunqiu/ds-code/internal/llm/deepseek"
	"github.com/hejunqiu/ds-code/internal/session"
	"github.com/hejunqiu/ds-code/internal/tool"
)

// Service builds API context and prepares requests.
type Service struct {
	Cfg        *config.Config
	Store      session.Store
	Tools      *tool.Registry
	AgentsMD   string
	AtExpander *AtExpander
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

// PrepareRequest builds the API view; compact is no-op in Phase 1–2.
func (s *Service) PrepareRequest(ctx context.Context, sessionID string) (*APIContextView, int, error) {
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

	toolDefs := s.Tools.Definitions()
	view := &APIContextView{
		SystemPrompt: defaultSystemBase,
		AgentsMD:     s.AgentsMD,
		GitSnapshot:  sess.GitSnapshot,
		ToolsJSON:    deepseek.ToolsJSON(toolDefs),
	}

	var apiMsgs []llm.Message
	for _, m := range msgs {
		if m.ID <= 0 {
			continue
		}
		// Phase 3+: filter by compact_up_to_message_id and inject compact summary.
		switch m.Role {
		case "user":
			apiMsgs = append(apiMsgs, llm.Message{Role: "user", Content: m.Content})
		case "assistant":
			am := llm.Message{
				Role:             "assistant",
				Content:          m.Content,
				ReasoningContent: m.ReasoningContent,
			}
			if m.ToolCallsJSON != "" {
				var calls []llm.ToolCall
				_ = json.Unmarshal([]byte(m.ToolCallsJSON), &calls)
				am.ToolCalls = calls
			}
			apiMsgs = append(apiMsgs, am)
		case "tool":
			apiMsgs = append(apiMsgs, llm.Message{
				Role:       "tool",
				Content:    m.Content,
				ToolCallID: m.ToolCallID,
				Name:       m.ToolName,
			})
		}
	}
	view.Messages = apiMsgs
	return view, nil
}

// CompactAPIContext is a no-op until Phase 3.
func (s *Service) CompactAPIContext(_ context.Context, _ string) error {
	return nil
}
