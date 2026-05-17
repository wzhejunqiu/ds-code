package config

import (
	"fmt"
	"slices"

	"github.com/spf13/viper"
)

var forbiddenYAMLKeys = []string{
	"llm.api_key",
	"session",
	"session.db_path",
	"audit.log_path",
	"checkpoint",
}

var allowedModels = []string{"deepseek-v4-pro", "deepseek-v4-flash"}
var allowedThinking = []string{"enabled", "disabled"}
var allowedEffort = []string{"high", "max"}
var allowedPermission = []string{"readonly", "ask", "auto"}
var allowedTruncateBy = []string{"chars", "tokenizer"}
var allowedRunMode = []string{"agent", "plan"}

// rejectForbiddenKeys blocks secrets and fixed paths from YAML (see docs/CONFIG.md).
func rejectForbiddenKeys(v *viper.Viper) error {
	for _, key := range forbiddenYAMLKeys {
		if v.IsSet(key) {
			return fmt.Errorf("config: forbidden key %q in YAML (see docs/CONFIG.md)", key)
		}
	}
	return nil
}

// validate checks enums and numeric bounds after merge.
func validate(cfg *Config) error {
	if !slices.Contains(allowedModels, cfg.LLM.Model) {
		return fmt.Errorf("config: llm.model must be one of %v, got %q", allowedModels, cfg.LLM.Model)
	}
	if cfg.LLM.MaxTokens < 1 || cfg.LLM.MaxTokens > cfg.Context.MaxOutputTokens {
		return fmt.Errorf(
			"config: llm.max_tokens must be between 1 and %d, got %d",
			cfg.Context.MaxOutputTokens,
			cfg.LLM.MaxTokens,
		)
	}
	if !slices.Contains(allowedThinking, cfg.LLM.Thinking.Type) {
		return fmt.Errorf("config: llm.thinking.type must be enabled or disabled, got %q", cfg.LLM.Thinking.Type)
	}
	if !slices.Contains(allowedEffort, cfg.LLM.ReasoningEffort) {
		return fmt.Errorf("config: llm.reasoning_effort must be high or max, got %q", cfg.LLM.ReasoningEffort)
	}
	if !slices.Contains(allowedPermission, cfg.Permission.Mode) {
		return fmt.Errorf("config: permission.mode must be readonly, ask, or auto, got %q", cfg.Permission.Mode)
	}
	if !slices.Contains(allowedTruncateBy, cfg.Context.TruncateBy) {
		return fmt.Errorf("config: context.truncate_by must be chars or tokenizer, got %q", cfg.Context.TruncateBy)
	}
	if cfg.Context.CompactThresholdRatio <= 0 || cfg.Context.CompactThresholdRatio > 1 {
		return fmt.Errorf("config: context.compact_threshold_ratio must be in (0,1], got %v", cfg.Context.CompactThresholdRatio)
	}
	if !slices.Contains(allowedRunMode, cfg.RunMode) {
		return fmt.Errorf("config: run_mode must be agent or plan, got %q", cfg.RunMode)
	}
	return nil
}
