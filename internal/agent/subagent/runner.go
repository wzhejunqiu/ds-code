// Package subagent runs a short read-only agent loop in an isolated session (see README.md).
package subagent

import (
	"context"
	"fmt"
	"io"

	"github.com/wzhejunqiu/ds-code/internal/agent"
	"github.com/wzhejunqiu/ds-code/internal/config"
	ctxpkg "github.com/wzhejunqiu/ds-code/internal/context"
	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/logging"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/session/subagentstore"
	"github.com/wzhejunqiu/ds-code/internal/tool"
	"go.uber.org/zap"
)

// RegisterFunc registers tools on a fresh registry.
type RegisterFunc func(reg *tool.Registry)

// Run executes a read-only sub-agent exploration and returns a summary string.
// run must already exist in subStore (created by the task tool). Optional cb streams
// nested tool events to the parent turn UI. maxTurns 0 uses min(8, cfg.Agent.MaxTurns).
func Run(ctx context.Context, cfg *config.Config, llmClient llm.Client, prompt string, register RegisterFunc, subStore subagentstore.Store, run subagentstore.Run, cb *agent.TurnCallbacks, maxTurns int) (string, error) {
	return executeRun(ctx, cfg, llmClient, prompt, register, subStore, run, cb, maxTurns)
}

func executeRun(ctx context.Context, cfg *config.Config, llmClient llm.Client, prompt string, register RegisterFunc, subStore subagentstore.Store, run subagentstore.Run, cb *agent.TurnCallbacks, maxTurns int) (string, error) {
	if prompt == "" {
		return "", fmt.Errorf("subagent: empty prompt")
	}
	if subStore == nil {
		return "", fmt.Errorf("subagent: store is required")
	}

	perm := permission.NewEngine("readonly", cfg.ProjectRoot, false)
	reg := tool.NewRegistry()
	register(reg)

	store := newSessionStore(subStore, run)
	sess, err := store.CreateSession(
		cfg.LLM.ResolveSubagentModel(),
		cfg.LLM.ResolveSubagentReasoningEffort(),
		cfg.LLM.ResolveSubagentThinkingType(),
		"readonly", "agent",
	)
	if err != nil {
		return "", err
	}

	agentsMD, _ := ctxpkg.LoadAgentsMD(cfg.ProjectRoot)
	rules, _ := ctxpkg.LoadRules(cfg.ProjectRoot)
	ctxSvc := &ctxpkg.Service{
		Cfg:      cfg,
		Store:    store,
		Tools:    reg,
		LLM:      llmClient,
		AgentsMD: agentsMD,
		Rules:    rules,
	}

	maxTurns = resolveMaxTurns(cfg, maxTurns)

	runner := &agent.Runner{
		LLM:      llmClient,
		Tools:    reg,
		Perm:     perm,
		Sessions: store,
		Context:  ctxSvc,
		Cfg:      cfg,
		MaxTurns: maxTurns,
		Out:      io.Discard,
	}

	logging.L().Debug("subagent run start",
		zap.String("subagent_session_id", sess.ID),
		zap.String("parent_run_id", run.ID),
		zap.Int("prompt_chars", len(prompt)),
	)
	result, err := runner.RunTurn(ctx, sess.ID, prompt, cb)
	if err != nil {
		logging.L().Debug("subagent run failed",
			zap.String("subagent_session_id", sess.ID),
			zap.Error(err),
		)
		return "", err
	}
	logging.L().Debug("subagent run done",
		zap.String("subagent_session_id", sess.ID),
		zap.Int("sub_rounds", result.SubRounds),
		zap.Int("prompt_tokens", result.Usage.PromptTokens),
	)
	summary := result.FinalContent
	if summary == "" {
		summary = result.FinalReasoning
	}
	return trimSummary(summary, cfg), nil
}

func resolveMaxTurns(cfg *config.Config, override int) int {
	if override > 0 {
		return override
	}
	maxTurns := 8
	if cfg.Agent.MaxTurns > 0 && cfg.Agent.MaxTurns < maxTurns {
		maxTurns = cfg.Agent.MaxTurns
	}
	return maxTurns
}

func trimSummary(s string, cfg *config.Config) string {
	max := cfg.Tools.Task.SummaryMaxChars
	if max <= 0 {
		max = 16_000
	}
	if len(s) <= max {
		return s
	}
	return s[:max] + agent.SubagentSummaryTruncated
}
