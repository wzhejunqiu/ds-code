package subagent

import (
	"context"
	"fmt"
	"io"

	"github.com/hejunqiu/ds-code/internal/agent"
	"github.com/hejunqiu/ds-code/internal/config"
	ctxpkg "github.com/hejunqiu/ds-code/internal/context"
	"github.com/hejunqiu/ds-code/internal/llm"
	"github.com/hejunqiu/ds-code/internal/permission"
	"github.com/hejunqiu/ds-code/internal/session"
	"github.com/hejunqiu/ds-code/internal/tool"
)

// RegisterFunc registers tools on a fresh registry.
type RegisterFunc func(reg *tool.Registry)

// Run executes a read-only sub-agent exploration and returns a summary string.
func Run(ctx context.Context, cfg *config.Config, llmClient llm.Client, prompt string, register RegisterFunc) (string, error) {
	if prompt == "" {
		return "", fmt.Errorf("subagent: empty prompt")
	}

	perm := permission.NewEngine("readonly", cfg.ProjectRoot, false)
	reg := tool.NewRegistry()
	register(reg)

	store := session.NewMemoryStore()
	sess, err := store.NewSession(cfg.LLM.Model, cfg.LLM.ReasoningEffort, cfg.LLM.Thinking.Type, "readonly", "agent")
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

	maxTurns := 8
	if cfg.Agent.MaxTurns > 0 && cfg.Agent.MaxTurns < maxTurns {
		maxTurns = cfg.Agent.MaxTurns
	}

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

	result, err := runner.RunTurn(ctx, sess.ID, prompt, nil)
	if err != nil {
		return "", err
	}
	summary := result.FinalContent
	if summary == "" {
		summary = result.FinalReasoning
	}
	return trimSummary(summary, cfg), nil
}

func trimSummary(s string, cfg *config.Config) string {
	max := cfg.Tools.Task.SummaryMaxChars
	if max <= 0 {
		max = 16_000
	}
	if len(s) <= max {
		return s
	}
	return s[:max] + "\n... [subagent summary truncated]"
}
