package spawn

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"
	"unicode/utf8"

	"github.com/wzhejunqiu/ds-code/internal/agent"
	"github.com/wzhejunqiu/ds-code/internal/config"
	ctxpkg "github.com/wzhejunqiu/ds-code/internal/context"
	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/logging"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/session"
	"github.com/wzhejunqiu/ds-code/internal/session/subagentstore"
	"github.com/wzhejunqiu/ds-code/internal/tool"
	"go.uber.org/zap"
)

// ExecuteRun creates a child Runner from a filtered tool pool, runs one turn,
// and returns a summary string. This is used for both Sync and Async paths.
func ExecuteRun(
	ctx context.Context,
	cfg *config.Config,
	llmClient llm.Client,
	run subagentstore.Run,
	def AgentTypeDefinition,
	parentPerm *permission.Engine,
	parentReg *tool.Registry,
	subStore subagentstore.Store,
	cb *agent.TurnCallbacks,
	maxTurns int,
) (string, error) {
	if run.Prompt == "" && run.SpawnKind != subagentstore.SpawnFork {
		return "", fmt.Errorf("spawn: empty prompt")
	}

	permMode := def.PermissionMode
	if permMode == "" {
		permMode = "inherit"
	}
	var perm *permission.Engine
	switch {
	case IsReadOnly(def) || permMode == "readonly":
		perm = permission.NewEngine("readonly", cfg.ProjectRoot, false)
	case permMode == "bubble", permMode == "inherit", permMode == "":
		// bubble: permission ask uses the parent's Prompter (TUI TUIPrompter when configured).
		perm = parentPerm
	default:
		perm = parentPerm
	}

	childReg := FilterToolRegistry(parentReg, def, run.Background)

	store := newSessionStore(subStore, run, permMode)
	sess, err := store.CreateSession(
		run.Model,
		cfg.LLM.ResolveSubagentReasoningEffort(),
		run.ThinkingType,
		permMode,
		"agent",
	)
	if err != nil {
		return "", err
	}

	var agentsMD, rules string
	if !def.OmitHeavyRules && run.SpawnKind != subagentstore.SpawnFork {
		agentsMD, _ = ctxpkg.LoadAgentsMD(cfg.ProjectRoot)
		rules, _ = ctxpkg.LoadRules(cfg.ProjectRoot)
	}

	ctxSvc := &ctxpkg.Service{
		Cfg:      cfg,
		Store:    store,
		Tools:    childReg,
		LLM:      llmClient,
		AgentsMD: agentsMD,
		Rules:    rules,
	}

	isFork := run.SpawnKind == subagentstore.SpawnFork
	if isFork {
		fc, ok := agent.ForkContextFromContext(ctx)
		if !ok {
			return "", fmt.Errorf("fork: missing parent context")
		}
		rendered := agent.RenderedSystemFromContext(ctx)
		forkMsgs := BuildForkMessages(fc.ParentMessages, fc.ParentToolCalls, buildChildDirective(run.Prompt))
		if err := seedForkMessages(ctx, store, sess.ID, forkMsgs); err != nil {
			return "", err
		}
		ctxSvc.ForkView = ctxpkg.BuildForkAPIContext(&ctxpkg.APIContextView{WindowTokens: cfg.Context.WindowTokens}, forkMsgs, rendered)
	} else {
		ctxSvc.AgentOverlay = SystemPromptOverlay(def)
	}

	if maxTurns <= 0 {
		maxTurns = 8
		if cfg.Agent.MaxTurns > 0 && cfg.Agent.MaxTurns < maxTurns {
			maxTurns = cfg.Agent.MaxTurns
		}
	}

	childRunner := &agent.Runner{
		LLM:      llmClient,
		Tools:    childReg,
		Perm:     perm,
		Sessions: store,
		Context:  ctxSvc,
		Cfg:      cfg,
		MaxTurns: maxTurns,
		Out:      io.Discard,
	}

	logging.L().Debug("spawn execute start",
		zap.String("run_id", run.ID),
		zap.String("agent_type", def.Type),
		zap.String("spawn_kind", string(run.SpawnKind)),
		zap.Int("prompt_chars", len(run.Prompt)),
	)

	var result *agent.TurnResult
	var runErr error
	if isFork {
		result, runErr = childRunner.RunTurnSeeded(ctx, sess.ID, cb)
	} else {
		result, runErr = childRunner.RunTurn(ctx, sess.ID, run.Prompt, cb)
	}
	if runErr != nil {
		logging.L().Debug("spawn execute failed",
			zap.String("run_id", run.ID),
			zap.Error(runErr),
		)
		return "", runErr
	}
	logging.L().Debug("spawn execute done",
		zap.String("run_id", run.ID),
		zap.Int("sub_rounds", result.SubRounds),
		zap.Int("prompt_tokens", result.Usage.PromptTokens),
	)

	summary := result.FinalContent
	if summary == "" {
		summary = result.FinalReasoning
	}
	return trimSummary(summary, cfg), nil
}

func seedForkMessages(ctx context.Context, store *sessionStore, sessionID string, msgs []llm.Message) error {
	for _, m := range msgs {
		sm := llmMessageToSession(m, sessionID)
		if err := store.AppendMessage(ctx, sm); err != nil {
			return err
		}
	}
	return nil
}

func llmMessageToSession(m llm.Message, sessionID string) session.Message {
	sm := session.Message{
		SessionID:        sessionID,
		Role:             m.Role,
		Content:          m.Content,
		ReasoningContent: m.ReasoningContent,
		ToolCallID:       m.ToolCallID,
		ToolName:         m.Name,
		CreatedAt:        time.Now().UTC(),
	}
	if len(m.ToolCalls) > 0 {
		b, _ := json.Marshal(m.ToolCalls)
		sm.ToolCallsJSON = string(b)
	}
	return sm
}

func resolveThinkingType(cfg *config.Config, def AgentTypeDefinition) string {
	if def.Type == "fork" {
		return cfg.LLM.ResolveSubagentThinkingType()
	}
	return "disabled"
}

func trimSummary(s string, cfg *config.Config) string {
	max := cfg.Tools.Agent.SummaryMaxChars
	if max <= 0 {
		max = 16_000
	}
	if len(s) <= max {
		return s
	}
	truncated := s[:max]
	for len(truncated) > 0 && len(truncated) >= max-3 {
		if utf8.ValidString(truncated) {
			break
		}
		truncated = truncated[:len(truncated)-1]
	}
	return truncated + agent.SubagentSummaryTruncated
}

// sessionStore adapts subagentstore.Store to session.Store for one agent run.
type sessionStore struct {
	sub      subagentstore.Store
	runID    string
	run      subagentstore.Run
	permMode string
}

func newSessionStore(sub subagentstore.Store, run subagentstore.Run, permMode string) *sessionStore {
	return &sessionStore{sub: sub, runID: run.ID, run: run, permMode: permMode}
}

func (s *sessionStore) CreateSession(model, effort, thinking, permMode, runMode string) (session.Session, error) {
	return s.toSession(), nil
}

func (s *sessionStore) NewSession(model, effort, thinking, permMode, runMode string) (session.Session, error) {
	return s.CreateSession(model, effort, thinking, permMode, runMode)
}

func (s *sessionStore) Create(_ context.Context, sess session.Session) error {
	return fmt.Errorf("agent session store: create not supported")
}

func (s *sessionStore) Get(ctx context.Context, _ string) (session.Session, error) {
	r, err := s.sub.GetRun(ctx, s.runID)
	if err != nil {
		return session.Session{}, err
	}
	s.run = r
	return s.toSession(), nil
}

func (s *sessionStore) ListMessages(ctx context.Context, _ string) ([]session.Message, error) {
	msgs, err := s.sub.ListMessages(ctx, s.runID)
	if err != nil {
		return nil, err
	}
	out := make([]session.Message, len(msgs))
	for i, m := range msgs {
		out[i] = subagentMessageToSession(m)
	}
	return out, nil
}

func (s *sessionStore) AppendMessage(ctx context.Context, msg session.Message) error {
	return s.sub.AppendMessage(ctx, sessionMessageToSubagent(msg, s.runID))
}

func (s *sessionStore) AddUsage(ctx context.Context, _ string, u llm.Usage) error {
	return s.sub.AddUsage(ctx, s.runID, u)
}

func (s *sessionStore) UpdateSession(ctx context.Context, _ string, fn func(*session.Session) error) error {
	sess := s.toSession()
	if err := fn(&sess); err != nil {
		return err
	}
	s.run.PromptTokensTotal = sess.PromptTokensTotal
	s.run.CompletionTokensTotal = sess.CompletionTokensTotal
	s.run.PromptCacheHitTokensTotal = sess.PromptCacheHitTokensTotal
	return nil
}

func (s *sessionStore) ListSessions(_ context.Context, _ int) ([]session.Summary, error) {
	return nil, fmt.Errorf("agent session store: list sessions not supported")
}

func (s *sessionStore) toSession() session.Session {
	return session.Session{
		ID:                        s.runID,
		Model:                     s.run.Model,
		ReasoningEffort:           s.run.ReasoningEffort,
		ThinkingType:              s.run.ThinkingType,
		PermissionMode:            s.permMode,
		RunMode:                   "agent",
		PromptTokensTotal:         s.run.PromptTokensTotal,
		CompletionTokensTotal:     s.run.CompletionTokensTotal,
		PromptCacheHitTokensTotal: s.run.PromptCacheHitTokensTotal,
		CreatedAt:                 s.run.CreatedAt,
		UpdatedAt:                 s.run.CreatedAt,
	}
}

func subagentMessageToSession(m subagentstore.Message) session.Message {
	return session.Message{
		ID:                   m.ID,
		SessionID:            m.RunID,
		Role:                 m.Role,
		Content:              m.Content,
		ReasoningContent:     m.ReasoningContent,
		ReasoningDurationMS:  m.ReasoningDurationMS,
		TurnDurationMS:       m.TurnDurationMS,
		ToolCallsJSON:        m.ToolCallsJSON,
		ToolCallID:           m.ToolCallID,
		ToolName:             m.ToolName,
		PromptTokens:         m.PromptTokens,
		CompletionTokens:     m.CompletionTokens,
		PromptCacheHitTokens: m.PromptCacheHitTokens,
		ModelID:              m.ModelID,
		PricingSnapshotJSON:  m.PricingSnapshotJSON,
		EstimatedCostCNY:     m.EstimatedCostCNY,
		CreatedAt:            m.CreatedAt,
	}
}

func sessionMessageToSubagent(m session.Message, runID string) subagentstore.Message {
	return subagentstore.Message{
		ID:                   m.ID,
		RunID:                runID,
		Role:                 m.Role,
		Content:              m.Content,
		ReasoningContent:     m.ReasoningContent,
		ReasoningDurationMS:  m.ReasoningDurationMS,
		TurnDurationMS:       m.TurnDurationMS,
		ToolCallsJSON:        m.ToolCallsJSON,
		ToolCallID:           m.ToolCallID,
		ToolName:             m.ToolName,
		PromptTokens:         m.PromptTokens,
		CompletionTokens:     m.CompletionTokens,
		PromptCacheHitTokens: m.PromptCacheHitTokens,
		ModelID:              m.ModelID,
		PricingSnapshotJSON:  m.PricingSnapshotJSON,
		EstimatedCostCNY:     m.EstimatedCostCNY,
		CreatedAt:            m.CreatedAt,
	}
}
