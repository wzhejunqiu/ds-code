package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/wzhejunqiu/ds-code/internal/agent"
	"github.com/wzhejunqiu/ds-code/internal/agent/spawn"
	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/session/subagentstore"
	"github.com/wzhejunqiu/ds-code/internal/tool"
	"github.com/wzhejunqiu/ds-code/internal/tool/builtin"
)

// AgentTool is the LLM-visible "agent" tool. It replaces the legacy "task" tool.
type AgentTool struct {
	Cfg    *config.Config
	Perm   *permission.Engine
	LLM    llm.Client
	Strict bool
	Store  subagentstore.Store
	Spawn  *spawn.Service
	sem    agentSemaphore
}

type agentSemaphore struct{ ch chan struct{} }

func newAgentSem(max int) agentSemaphore {
	if max <= 0 {
		max = 3
	}
	return agentSemaphore{ch: make(chan struct{}, max)}
}

func (s agentSemaphore) acquire(ctx context.Context) error {
	select {
	case s.ch <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s agentSemaphore) release() { <-s.ch }

// NewAgentTool creates an agent tool wired to the spawn service.
func NewAgentTool(cfg *config.Config, perm *permission.Engine, llmClient llm.Client, strict bool, store subagentstore.Store, parentReg *tool.Registry) *AgentTool {
	return &AgentTool{
		Cfg:    cfg,
		Perm:   perm,
		LLM:    llmClient,
		Strict: strict,
		Store:  store,
		Spawn:  spawn.NewService(cfg, perm, parentReg, llmClient, store),
		sem:    newAgentSem(cfg.Tools.Agent.MaxParallel),
	}
}

func (t *AgentTool) Name() string { return tool.NameAgent.String() }

func (t *AgentTool) Description() string {
	return RenderDesc(t.Cfg.Tools.Agent.MaxParallel)
}

func (t *AgentTool) Schema() map[string]any {
	types := t.Spawn.Registry.ListToolTypes()
	enum := make([]any, len(types))
	for i, typ := range types {
		enum[i] = typ
	}

	return tool.ObjectSchema(map[string]any{
		"description":       map[string]any{"type": "string", "description": SchemaAgentDescription},
		"prompt":            map[string]any{"type": "string", "description": SchemaAgentPrompt},
		"subagent_type":     map[string]any{"type": "string", "description": SchemaAgentType, "enum": enum},
		"run_in_background": map[string]any{"type": "boolean", "description": SchemaAgentBackground},
	}, []string{"description", "prompt"}, t.Strict)
}

func (t *AgentTool) PermissionLevel() permission.Level { return permission.LevelLow }

// SpawnService returns the underlying spawn service (for wiring notifications).
func (t *AgentTool) SpawnService() *spawn.Service { return t.Spawn }

func (t *AgentTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var params spawn.Params
	if err := json.Unmarshal(args, &params); err != nil {
		return "", err
	}
	if params.Prompt == "" {
		return "", fmt.Errorf("%s", builtin.ErrPromptRequired)
	}

	inv, ok := agent.ToolInvocationFromContext(ctx)
	if !ok || inv.SessionID == "" || inv.ToolCallID == "" {
		return "", fmt.Errorf("%s", ErrMissingParent)
	}
	if t.Store == nil {
		return "", fmt.Errorf("%s", ErrNoStore)
	}

	if err := t.sem.acquire(ctx); err != nil {
		return "", err
	}
	defer t.sem.release()

	return t.Spawn.Handle(ctx, inv, params, t.Perm.Interactive)
}
