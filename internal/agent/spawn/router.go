package spawn

import (
	"context"
	"fmt"

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
	// SubagentType selects a built-in agent definition; empty triggers fork when enabled.
	SubagentType string `json:"subagent_type,omitempty"`
	// Model optionally overrides the agent type or global subagent model.
	Model string `json:"model,omitempty"`
	// RunInBackground requests async execution (not supported on fork path).
	RunInBackground bool `json:"run_in_background,omitempty"`
	// Isolation selects workspace isolation; "worktree" is limited to general-purpose.
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
	Model       string
	Isolation   string
}

// Route resolves the spawn path based on params, type definition, and config.
//
// Decision tree:
//  1. subagent_type omitted + fork_enabled + interactive → Fork
//  2. force_background (verification) → Async
//  3. run_in_background → Async
//  4. Default → Sync
func Route(ctx context.Context, params Params, inv agent.ToolInvocation, reg *Registry, cfg *config.Config, interactive bool) (RouteDecision, error) {
	// Empty subagent_type falls back to general-purpose for tool pool and overlays.
	def, err := reg.Resolve(params.SubagentType)
	if err != nil {
		return RouteDecision{}, err
	}

	// Fork path: type omitted + fork enabled + interactive session.
	if params.SubagentType == "" && cfg.Tools.Agent.ForkEnabled && interactive {
		// Fork children inherit QuerySourceFork and must not spawn another fork.
		if QuerySourceFromContext(ctx) == QuerySourceFork {
			return RouteDecision{}, fmt.Errorf("fork: cannot fork from fork child")
		}
		// Detect recursive fork via fork-boilerplate tag in parent messages.
		if fc, ok := agent.ForkContextFromContext(ctx); ok && IsInForkChild(fc.ParentMessages) {
			return RouteDecision{}, fmt.Errorf("fork: recursive fork detected — fork children cannot spawn fork sub-agents")
		}
		if params.RunInBackground {
			return RouteDecision{}, fmt.Errorf("fork: run_in_background is not supported")
		}
		return RouteDecision{
			SpawnKind:   subagentstore.SpawnFork,
			IsFork:      true,
			Background:  params.RunInBackground,
			Definition:  def,
			Description: params.Description,
			Prompt:      params.Prompt,
			Model:       params.Model,
			Isolation:   params.Isolation,
		}, nil
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
		Model:       params.Model,
		Isolation:   params.Isolation,
	}, nil
}
