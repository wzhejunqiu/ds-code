package spawn

import (
	"context"

	"github.com/wzhejunqiu/ds-code/internal/agent"
	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/session/subagentstore"
)

// Params is the parsed agent tool arguments from the LLM.
type Params struct {
	// Description is a short label for UI and tool return formatting.
	Description string `json:"description"`
	// Prompt is the sub-agent task directive.
	Prompt string `json:"prompt"`
	// SubagentType selects a built-in agent definition; empty defaults to general-purpose.
	SubagentType string `json:"subagent_type,omitempty"`
	// RunInBackground requests async execution.
	RunInBackground bool `json:"run_in_background,omitempty"`
	// Isolation selects workspace isolation; "worktree" is limited to general-purpose (not LLM-exposed).
	Isolation string `json:"isolation,omitempty"`
}

// RouteDecision is the outcome of routing an agent spawn.
type RouteDecision struct {
	// SpawnKind records sync, async, or fork for persistence and telemetry.
	SpawnKind subagentstore.SpawnKind
	// Background means the parent tool returns immediately with async_launched.
	Background bool
	// IsFork selects the fork message-construction path (shared parent prompt cache).
	IsFork bool
	// Definition is the resolved agent type (tools, overlays, readonly mode).
	Definition  AgentTypeDefinition
	Description string
	Prompt      string
	Isolation   string
}

// Route resolves the spawn path based on params, type definition, and config.
//
// Decision tree:
//  1. force_background (verification) → Async
//  2. run_in_background → Async
//  3. Default → Sync
func Route(ctx context.Context, params Params, inv agent.ToolInvocation, reg *Registry, cfg *config.Config, interactive bool) (RouteDecision, error) {
	_ = ctx
	_ = inv
	_ = cfg
	_ = interactive

	// Empty subagent_type falls back to general-purpose for tool pool and overlays.
	def, err := reg.Resolve(params.SubagentType)
	if err != nil {
		return RouteDecision{}, err
	}

	// Async path: verification types force background, or caller opts in explicitly.
	background := def.ForceBackground || params.RunInBackground

	spawnKind := subagentstore.SpawnSync
	if background {
		spawnKind = subagentstore.SpawnAsync
	}

	return RouteDecision{
		SpawnKind:   spawnKind,
		Background:  background,
		Definition:  def,
		Description: params.Description,
		Prompt:      params.Prompt,
		Isolation:   params.Isolation,
	}, nil
}
