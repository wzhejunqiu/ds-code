package config

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wzhejunqiu/ds-code/internal/permissionmode"
	"github.com/wzhejunqiu/ds-code/internal/runmode"
)

// applyChangedFlags merges only CLI flags the user explicitly set (highest priority).
func applyChangedFlags(cfg *Config, cmd *cobra.Command) error {
	fs := cmd.PersistentFlags()

	if f := fs.Lookup("model"); f != nil && f.Changed {
		cfg.LLM.Model = f.Value.String()
	}
	if f := fs.Lookup("subagent-model"); f != nil && f.Changed {
		cfg.LLM.Subagent.Model = f.Value.String()
	}
	if f := fs.Lookup("max-tokens"); f != nil && f.Changed {
		v, err := fs.GetInt("max-tokens")
		if err != nil {
			return err
		}
		cfg.LLM.MaxTokens = v
	}
	if f := fs.Lookup("thinking"); f != nil && f.Changed {
		cfg.LLM.Thinking.Type = f.Value.String()
	}
	if f := fs.Lookup("subagent-thinking"); f != nil && f.Changed {
		cfg.LLM.Subagent.Thinking.Type = f.Value.String()
	}
	if f := fs.Lookup("reasoning-effort"); f != nil && f.Changed {
		cfg.LLM.ReasoningEffort = f.Value.String()
	}
	if f := fs.Lookup("subagent-reasoning-effort"); f != nil && f.Changed {
		cfg.LLM.Subagent.ReasoningEffort = f.Value.String()
	}
	if f := fs.Lookup("strict-tools"); f != nil && f.Changed {
		v, err := fs.GetBool("strict-tools")
		if err != nil {
			return err
		}
		cfg.LLM.StrictTools = v
	}
	if f := fs.Lookup("permission-mode"); f != nil && f.Changed {
		m, err := permissionmode.Parse(f.Value.String())
		if err != nil {
			return fmt.Errorf("config: permission-mode: %w", err)
		}
		cfg.Permission.Mode = m
	}
	if f := fs.Lookup("dangerously-auto"); f != nil && f.Changed {
		v, err := fs.GetBool("dangerously-auto")
		if err != nil {
			return err
		}
		if v {
			cfg.Permission.Mode = permissionmode.Auto
		}
	}
	if f := fs.Lookup("audit-log"); f != nil && f.Changed {
		v, err := fs.GetBool("audit-log")
		if err != nil {
			return err
		}
		cfg.Audit.Enabled = v
	}
	if f := fs.Lookup("plan"); f != nil && f.Changed {
		v, err := fs.GetBool("plan")
		if err != nil {
			return err
		}
		if v {
			cfg.RunMode = runmode.Plan
		}
	}
	if f := fs.Lookup("trace"); f != nil && f.Changed {
		v, err := fs.GetBool("trace")
		if err != nil {
			return err
		}
		cfg.Tracing.Enabled = v
	}
	if f := fs.Lookup("trace-exporter"); f != nil && f.Changed {
		cfg.Tracing.Exporter = f.Value.String()
	}
	if f := fs.Lookup("trace-otlp-endpoint"); f != nil && f.Changed {
		cfg.Tracing.OTLPEndpoint = f.Value.String()
	}
	return nil
}

// ApplyCLIDerived sets runtime fields from flags (prompt, json output, verbosity).
func ApplyCLIDerived(cfg *Config, cmd *cobra.Command) error {
	fs := cmd.PersistentFlags()
	prompt, err := fs.GetString("prompt")
	if err != nil {
		return fmt.Errorf("config: prompt flag: %w", err)
	}
	cfg.Prompt = prompt
	jsonOut, err := fs.GetBool("json")
	if err != nil {
		return fmt.Errorf("config: json flag: %w", err)
	}
	cfg.JSONOutput = jsonOut
	verbose, err := fs.GetCount("verbose")
	if err != nil {
		return fmt.Errorf("config: verbose flag: %w", err)
	}
	cfg.LogVerbosity = verbose
	allowSensitive, err := fs.GetBool("allow-log-sensitive-data")
	if err != nil {
		return fmt.Errorf("config: allow-log-sensitive-data flag: %w", err)
	}
	cfg.AllowLogSensitiveData = cfg.LogVerbosity >= 2 && allowSensitive
	return nil
}
