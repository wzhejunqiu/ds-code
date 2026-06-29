package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/wzhejunqiu/ds-code/internal/datadir"
	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/logging"
	"github.com/wzhejunqiu/ds-code/internal/role"
	"go.uber.org/zap"
)

// EphemeralOpts configures a /btw side-channel request.
type EphemeralOpts struct {
	SessionID          string
	IncludeRecentTurns int
	MaxTokens          int
	CountTowardSession bool
	MergedSystem       string
}

// EphemeralResult is the outcome of RunEphemeral.
type EphemeralResult struct {
	Content   string
	Reasoning string
	Usage     llm.Usage
}

// RunEphemeral handles /btw: no tools, no history writes (S13).
func (r *Runner) RunEphemeral(ctx context.Context, prompt string, opts EphemeralOpts) (*EphemeralResult, error) {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return nil, fmt.Errorf("btw: empty prompt")
	}
	maxTokens := opts.MaxTokens
	if maxTokens <= 0 {
		maxTokens = r.Cfg.BTW.MaxTokens
	}
	if maxTokens <= 0 {
		maxTokens = 4096
	}

	var messages []llm.Message
	if opts.IncludeRecentTurns > 0 && opts.SessionID != "" && r.Context != nil {
		view, err := r.Context.BuildAPIContext(ctx, opts.SessionID)
		if err == nil && len(view.Messages) > 0 {
			messages = append(messages, view.Messages...)
		}
	}
	messages = append(messages, llm.Message{Role: role.User, Content: prompt})

	system := opts.MergedSystem
	if system == "" {
		system = BTWDefaultSystem
		if r.Context != nil && r.Context.AgentsMD != "" {
			system += BTWAgentsHeader + r.Context.AgentsMD
		}
	}

	sess, err := r.Sessions.Get(ctx, opts.SessionID)
	if err != nil {
		return nil, err
	}

	userID := datadir.Identifier()
	logging.L().Debug("ephemeral request",
		zap.String("session_id", opts.SessionID),
		zap.Int("messages", len(messages)),
		zap.String("user_id", userID),
	)
	resp, err := r.LLM.Chat(ctx, llm.Request{
		MergedSystem:    system,
		Messages:        messages,
		Model:           sess.Model,
		MaxTokens:       maxTokens,
		Stream:          false,
		ThinkingType:    "disabled",
		ReasoningEffort: sess.ReasoningEffort,
		UserID:          userID,
		StrictTools:     r.Cfg.LLM.StrictTools,
	})
	if err != nil {
		return nil, err
	}

	if opts.CountTowardSession && opts.SessionID != "" {
		_ = r.Sessions.AddUsage(ctx, opts.SessionID, resp.Usage)
	}

	return &EphemeralResult{
		Content:   resp.Content,
		Reasoning: resp.ReasoningContent,
		Usage:     resp.Usage,
	}, nil
}
