package spawn

import (
	"context"
	"fmt"

	"github.com/wzhejunqiu/ds-code/internal/agent"
	ctxpkg "github.com/wzhejunqiu/ds-code/internal/context"
	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/role"
	"github.com/wzhejunqiu/ds-code/internal/session/subagentstore"
)

// ErrSkillNotFork indicates the skill does not request fork context.
var ErrSkillNotFork = fmt.Errorf("skill is not a fork skill")

// FromSkill runs a skill with context:fork via the fork spawn path (no agent tool params).
func (s *Service) FromSkill(ctx context.Context, inv agent.ToolInvocation, skillName string, interactive bool) (string, error) {
	meta, body, err := ctxpkg.LoadSkillWithMeta(s.Cfg.ProjectRoot, skillName)
	if err != nil {
		return "", err
	}
	if meta.ContextMode != "fork" {
		return "", ErrSkillNotFork
	}
	if !s.Cfg.Tools.Agent.ForkEnabled {
		return "", fmt.Errorf("fork is disabled in config (tools.agent.fork_enabled)")
	}
	if !interactive {
		return "", fmt.Errorf("skill fork requires interactive mode")
	}
	def, err := s.Registry.Resolve("")
	if err != nil {
		return "", err
	}
	def.Type = AgentTypeFork
	decision := RouteDecision{
		SpawnKind:   subagentstore.SpawnFork,
		IsFork:      true,
		Background:  false,
		Definition:  def,
		Description: "skill:" + skillName,
		Prompt:      body,
	}
	model := ResolveModel("", def.Model, s.Cfg)
	run, err := s.Store.CreateRun(ctx, subagentstore.CreateRunParams{
		ParentSessionID:  inv.SessionID,
		ParentToolCallID: inv.ToolCallID,
		AgentType:        AgentTypeFork.String(),
		SpawnKind:        subagentstore.SpawnFork,
		Label:            truncateLabel(decision.Description, body, 48),
		Prompt:           body,
		Model:            model,
		ReasoningEffort:  s.Cfg.LLM.ResolveSubagentReasoningEffort(),
		ThinkingType:     resolveThinkingType(s.Cfg, def, s.parentSessionThinking(ctx, inv.SessionID), true),
	})
	if err != nil {
		return "", err
	}
	if s.ParentContext != nil {
		view, err := s.ParentContext.BuildAPIContext(ctx, inv.SessionID)
		if err == nil {
			ctx = agent.WithRenderedSystem(ctx, view.MergedSystem())
			var parentCalls []llm.ToolCall
			for i := len(view.Messages) - 1; i >= 0; i-- {
				m := view.Messages[i]
				if m.Role == role.Assistant && len(m.ToolCalls) > 0 {
					parentCalls = m.ToolCalls
					break
				}
			}
			ctx = agent.WithForkContext(ctx, agent.ForkContext{
				ParentMessages:  view.Messages,
				ParentToolCalls: parentCalls,
			})
		}
	}
	ctx = WithQuerySource(ctx, QuerySourceSkill)

	parent := agent.TurnCallbacksFromContext(ctx)
	if s.Hooks != nil {
		s.Hooks.Run(ctx, agent.HookSubagentStart, agent.MarshalHookInput(agent.HookInput{
			SessionID: inv.SessionID,
			AgentID:   run.ID,
			AgentType: AgentTypeFork.String(),
		}))
	}
	return s.runSync(ctx, inv, run, decision, parent)
}
