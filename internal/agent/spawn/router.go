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
	Description     string `json:"description"`
	Prompt          string `json:"prompt"`
	SubagentType    string `json:"subagent_type,omitempty"`
	Model           string `json:"model,omitempty"`
	RunInBackground bool   `json:"run_in_background,omitempty"`
	Isolation       string `json:"isolation,omitempty"`
}

// RouteDecision is the outcome of routing an agent spawn.
type RouteDecision struct {
	SpawnKind   subagentstore.SpawnKind
	Background  bool
	IsFork      bool
	Definition  AgentTypeDefinition
	Description string
	Prompt      string
	Model       string
	Isolation   string
}

// Route resolves the spawn path based on params, type definition, and config.
// Decision tree:
// 1. subagent_type omitted + fork_enabled + interactive → Fork
// 2. force_background (verification) → Async
// 3. run_in_background → Async
// 4. Default → Sync
func Route(ctx context.Context, params Params, inv agent.ToolInvocation, reg *Registry, cfg *config.Config, interactive bool) (RouteDecision, error) {
	def, err := reg.Resolve(params.SubagentType)
	if err != nil {
		return RouteDecision{}, err
	}

	// Fork path: type omitted + fork enabled + interactive
	if params.SubagentType == "" && cfg.Tools.Agent.ForkEnabled && interactive {
		if QuerySourceFromContext(ctx) == QuerySourceFork {
			return RouteDecision{}, fmt.Errorf("fork: cannot fork from fork child")
		}
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

	// Async path: verification forced, or explicit background request
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
