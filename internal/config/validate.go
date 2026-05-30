package config

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

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
	if cfg.LLM.Subagent.Model != "" && !slices.Contains(allowedModels, cfg.LLM.Subagent.Model) {
		return fmt.Errorf("config: llm.subagent.model must be one of %v, got %q", allowedModels, cfg.LLM.Subagent.Model)
	}
	if cfg.LLM.Subagent.Thinking.Type != "" && !slices.Contains(allowedThinking, cfg.LLM.Subagent.Thinking.Type) {
		return fmt.Errorf("config: llm.subagent.thinking.type must be enabled or disabled, got %q", cfg.LLM.Subagent.Thinking.Type)
	}
	if cfg.LLM.Subagent.ReasoningEffort != "" && !slices.Contains(allowedEffort, cfg.LLM.Subagent.ReasoningEffort) {
		return fmt.Errorf("config: llm.subagent.reasoning_effort must be high or max, got %q", cfg.LLM.Subagent.ReasoningEffort)
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
	if cfg.Context.CollapseThresholdRatio <= 0 || cfg.Context.CollapseThresholdRatio > 1 {
		return fmt.Errorf("config: context.collapse_threshold_ratio must be in (0,1], got %v", cfg.Context.CollapseThresholdRatio)
	}
	if cfg.Context.SnipKeepRounds < 0 {
		return fmt.Errorf("config: context.snip_keep_rounds must be >= 0, got %d", cfg.Context.SnipKeepRounds)
	}
	if !cfg.RunMode.Configured() {
		return fmt.Errorf("config: run_mode must be agent or plan, got %q", cfg.RunMode)
	}
	compiled, err := compileEnvBlacklist(cfg.Tools.Shell.EnvBlacklist)
	if err != nil {
		return err
	}
	cfg.Tools.Shell.EnvBlacklistCompiled = compiled
	return nil
}

func compileEnvBlacklist(patterns []string) ([]*regexp.Regexp, error) {
	if len(patterns) == 0 {
		return nil, nil
	}
	out := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		re, err := regexp.Compile(p)
		if err != nil {
			return nil, fmt.Errorf("config: tools.shell.env_blacklist: invalid pattern %q: %w", p, err)
		}
		out = append(out, re)
	}
	return out, nil
}
