package spawn

import "github.com/wzhejunqiu/ds-code/internal/config"

// ResolveModel picks the model for a child agent using the priority chain:
// 1. Agent type definition override (unless "inherit")
// 2. Config llm.subagent.model (when set, including built-in default)
// 3. Parent session model
// 4. Fall back to llm.model when parent model is also empty
func ResolveModel(defModel ModelSelection, cfg *config.Config, parentModel string) string {
	if !defModel.Inherit() {
		return resolveAlias(defModel.String(), cfg)
	}
	if cfg.LLM.Subagent.Model != "" {
		return cfg.LLM.Subagent.Model
	}
	if parentModel != "" {
		return parentModel
	}
	return cfg.LLM.Model
}

// resolveSubagentMaxTurns returns the max sub-rounds for a child agent runner.
func resolveSubagentMaxTurns(cfg *config.Config) int {
	if cfg.LLM.Subagent.MaxTurns > 0 {
		return cfg.LLM.Subagent.MaxTurns
	}
	return 500
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
