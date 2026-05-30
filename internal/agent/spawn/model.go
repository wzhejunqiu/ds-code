package spawn

import "github.com/wzhejunqiu/ds-code/internal/config"

// ResolveModel picks the model for a child agent using the priority chain:
// 1. Agent tool parameter override
// 2. Agent type definition override (unless "inherit")
// 3. Config llm.subagent.model
// 4. Fall back to main model
func ResolveModel(paramsModel string, defModel ModelSelection, cfg *config.Config) string {
	if paramsModel != "" {
		return resolveAlias(paramsModel, cfg)
	}
	if !defModel.Inherit() {
		return resolveAlias(defModel.String(), cfg)
	}
	if cfg.LLM.Subagent.Model != "" {
		return cfg.LLM.Subagent.Model
	}
	return cfg.LLM.Model
}

// resolveSubagentMaxTurns returns the max sub-rounds for a child agent runner.
func resolveSubagentMaxTurns(cfg *config.Config) int {
	if cfg.LLM.Subagent.MaxTurns > 0 {
		return cfg.LLM.Subagent.MaxTurns
	}
	return 8
}

func resolveAlias(alias string, cfg *config.Config) string {
	switch alias {
	case "sonnet", "opus", "haiku":
		return cfg.LLM.Model
	case "":
		return cfg.LLM.Model
	default:
		return alias
	}
}
