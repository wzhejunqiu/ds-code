package spawn

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/wzhejunqiu/ds-code/internal/agent"
	"github.com/wzhejunqiu/ds-code/internal/config"
	ctxpkg "github.com/wzhejunqiu/ds-code/internal/context"
	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/logging"
	"github.com/wzhejunqiu/ds-code/internal/mcp/resultstore"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/permissionmode"
	"github.com/wzhejunqiu/ds-code/internal/session"
	"github.com/wzhejunqiu/ds-code/internal/session/subagentstore"
	"github.com/wzhejunqiu/ds-code/internal/tool"
	"go.uber.org/zap"
)

// ExecuteRun assembles a restricted child Runner, runs one turn, and returns a summary.
// Used by both sync and async spawn paths.
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
	hooks *agent.HookManager,
	maxTurns int,
	mcpResults *resultstore.Store,
) (string, error) {
	if run.Prompt == "" && run.SpawnKind != subagentstore.SpawnFork {
		return "", fmt.Errorf("spawn: empty prompt")
	}

	// Worktree isolation runs tools against the detached checkout, not project root.
	workspace := cfg.ProjectRoot
	if run.WorktreePath != "" {
		workspace = run.WorktreePath
	}

	permMode := def.PermissionMode
	if permMode == "" {
		permMode = AgentPermModeInherit
	}
	var perm *permission.Engine
	switch {
	case IsReadOnly(def) || permMode == AgentPermModeReadonly:
		perm = permission.NewEngine(permissionmode.Readonly, workspace, false)
		perm.ProjectRoot = cfg.ProjectRoot
		copyWebPermFields(perm, parentPerm, cfg)
	case permMode == AgentPermModeBubble, permMode == AgentPermModeInherit:
		// bubble: permission ask uses the parent's Prompter (TUI TUIPrompter when configured).
		if run.WorktreePath != "" {
			// Rebind workspace while keeping parent Prompter for bubble-up asks.
			perm = permission.NewEngine(parentPerm.Mode, workspace, parentPerm.Interactive)
			perm.Prompter = parentPerm.Prompter
			perm.ProjectRoot = cfg.ProjectRoot
			copyWebPermFields(perm, parentPerm, cfg)
		} else {
			perm = parentPerm
		}
	default:
		perm = parentPerm
	}

	// Layer 1 always strips agent; background runs get an async tool whitelist.
	childReg := FilterToolRegistry(parentReg, def, run.Background)
	if run.WorktreePath != "" {
		childReg = tool.RebindRegistryPerm(childReg, perm)
	}

	// Single-run adapter: subagentstore rows masquerade as a session.Store.
	store := newSessionStore(subStore, run, permMode)
	sess, err := store.CreateSession(
		run.Model,
		cfg.LLM.ResolveSubagentReasoningEffort(),
		run.ThinkingType,
		permMode.ToSessionMode(),
		session.RunModeAgent,
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
		Cfg:        cfg,
		Store:      store,
		Tools:      childReg,
		LLM:        llmClient,
		AgentsMD:   agentsMD,
		Rules:      rules,
		AtExpander: &ctxpkg.AtExpander{Cfg: cfg, Perm: perm},
	}

	isFork := run.SpawnKind == subagentstore.SpawnFork
	if isFork {
		// Fork shares parent API prefix; only the trailing user directive differs.
		fc, ok := agent.ForkContextFromContext(ctx)
		if !ok {
			return "", fmt.Errorf("fork: missing parent context")
		}
		rendered := agent.RenderedSystemFromContext(ctx)
		if mem := FormatAgentMemory(AgentTypeFork.String()); mem != "" {
			rendered = rendered + "\n" + mem
		}
		forkMsgs := BuildForkMessages(fc.ParentMessages, fc.ParentToolCalls, run.Prompt)
		if err := seedForkMessages(ctx, store, sess.ID, forkMsgs); err != nil {
			return "", err
		}
		ctxSvc.ForkView = ctxpkg.BuildForkAPIContext(&ctxpkg.APIContextView{WindowTokens: cfg.Context.WindowTokens}, forkMsgs, rendered)
	} else {
		overlay := SystemPromptOverlay(def)
		if mem := FormatAgentMemory(def.Type.String()); mem != "" {
			overlay = overlay + "\n" + mem
		}
		ctxSvc.AgentOverlay = overlay
		if def.Type == AgentTypeVerification {
			ctxSvc.VerificationMode = true
		}
	}

	if maxTurns <= 0 {
		maxTurns = resolveSubagentMaxTurns(cfg)
	}

	childRunner := &agent.Runner{
		LLM:         llmClient,
		Tools:       childReg,
		Perm:        perm,
		Sessions:    store,
		Context:     ctxSvc,
		Cfg:         cfg,
		MaxTurns:    maxTurns,
		Out:         io.Discard,
		Hooks:       hooks,
		ForSubagent: true,
		MCPResults:  mcpResults,
	}

	logging.L().Debug("spawn execute start",
		zap.String("run_id", run.ID),
		zap.String("agent_type", def.Type.String()),
		zap.String("spawn_kind", string(run.SpawnKind)),
		zap.Int("prompt_chars", len(run.Prompt)),
	)

	var result *agent.TurnResult
	var runErr error
	if isFork {
		ctx = WithQuerySource(ctx, QuerySourceFork)
		// Messages already seeded; RunTurnSeeded continues from fork prefix.
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

	// Prefer visible assistant content; fall back to reasoning when content is empty.
	summary := result.FinalContent
	if summary == "" {
		summary = result.FinalReasoning
	}
	return trimSummary(summary, cfg), nil
}

// seedForkMessages persists pre-built fork API messages before RunTurnSeeded.
func seedForkMessages(ctx context.Context, store *sessionStore, sessionID string, msgs []llm.Message) error {
	for _, m := range msgs {
		sm := llmMessageToSession(m, sessionID)
		if err := store.AppendMessage(ctx, sm); err != nil {
			return err
		}
	}
	return nil
}

// llmMessageToSession converts a fork seed message into a session.Message row.
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

// resolveThinkingType picks thinking mode for a new run.
// Fork inherits the parent session; other sub-agents default to disabled.
func resolveThinkingType(cfg *config.Config, def AgentTypeDefinition, parentThinking string, isFork bool) string {
	if isFork || def.Type == AgentTypeFork {
		if parentThinking != "" {
			return parentThinking
		}
		if cfg.LLM.Thinking.Type != "" {
			return cfg.LLM.Thinking.Type
		}
		return "enabled"
	}
	return "disabled"
}

// trimSummary caps summary length and appends a truncation marker when clipped.
func trimSummary(s string, cfg *config.Config) string {
	max := summaryMaxChars(cfg)
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	var b strings.Builder
	n := 0
	for _, r := range s {
		if n >= max {
			break
		}
		b.WriteRune(r)
		n++
	}
	return b.String() + agent.SubagentSummaryTruncated
}

// sessionStore adapts subagentstore.Store to session.Store for one agent run.
// All session IDs map to a single subagent Run row.
type sessionStore struct {
	sub   subagentstore.Store // underlying sub-agent persistence
	runID string              // fixed run ID for all session operations
	run   subagentstore.Run   // cached run metadata for token totals
	// Agent-level perm mode; mapped to session.PermissionMode in toSession when applicable.
	permMode AgentPermMode
}

func newSessionStore(sub subagentstore.Store, run subagentstore.Run, permMode AgentPermMode) *sessionStore {
	return &sessionStore{sub: sub, runID: run.ID, run: run, permMode: permMode}
}

// CreateSession returns a synthetic session backed by the subagent run metadata.
func (s *sessionStore) CreateSession(model, effort, thinking string, permMode session.PermissionMode, runMode session.RunMode) (session.Session, error) {
	return s.toSession(), nil
}

func (s *sessionStore) NewSession(model, effort, thinking string, permMode session.PermissionMode, runMode session.RunMode) (session.Session, error) {
	return s.CreateSession(model, effort, thinking, permMode, runMode)
}

func (s *sessionStore) Create(_ context.Context, sess session.Session) error {
	return fmt.Errorf("agent session store: create not supported")
}

// Get refreshes run metadata from subagentstore and projects it as a session.
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

// UpdateSession mirrors token totals from Runner back onto the in-memory run snapshot.
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
		PermissionMode:            s.permMode.ToSessionMode(),
		RunMode:                   session.RunModeAgent,
		PromptTokensTotal:         s.run.PromptTokensTotal,
		CompletionTokensTotal:     s.run.CompletionTokensTotal,
		PromptCacheHitTokensTotal: s.run.PromptCacheHitTokensTotal,
		CreatedAt:                 s.run.CreatedAt,
		UpdatedAt:                 s.run.CreatedAt,
	}
}

// subagentMessageToSession maps a subagent transcript row to session.Message.
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

func copyWebPermFields(dst, parent *permission.Engine, cfg *config.Config) {
	if parent != nil && len(parent.WebAllowlist) > 0 {
		dst.WebAllowlist = append([]string(nil), parent.WebAllowlist...)
	} else {
		dst.WebAllowlist = append([]string(nil), cfg.Web.Allowlist...)
	}
	if parent != nil {
		dst.WebFetchPrompter = parent.WebFetchPrompter
	}
}

// sessionMessageToSubagent maps a session.Message row back to subagentstore.Message.
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
