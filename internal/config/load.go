package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Options controls config loading outside YAML merge.
type Options struct {
	// StartDir is used to resolve project_root (default: os.Getwd).
	StartDir string
	// RequireAPIKey when true fails if no API key is set (agent phases).
	RequireAPIKey bool
	// SkipProjectDataDir skips mkdir under ~/.ds-code (tests only).
	SkipProjectDataDir bool
}

// Load merges defaults, user YAML, project YAML, env (DS_CODE_*), and cobra flags into Config.
// See README.md for the full pipeline.
func Load(cmd *cobra.Command, opts Options) (*Config, error) {
	if opts.StartDir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		opts.StartDir = wd
	}

	v := viper.New()
	setDefaults(v)
	v.SetEnvPrefix("DS_CODE")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := readYAMLIfExists(v, UserConfigPath); err != nil {
		return nil, err
	}

	projectRoot, err := ResolveProjectRoot(opts.StartDir)
	if err != nil {
		return nil, err
	}
	if err := rejectProjectBilling(projectRoot); err != nil {
		return nil, err
	}
	projectYAML := ProjectConfigPath(projectRoot)
	if err := readYAMLIfExists(v, func() (string, error) { return projectYAML, nil }); err != nil {
		return nil, err
	}

	if err := rejectForbiddenKeys(v); err != nil {
		return nil, err
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("config: unmarshal: %w", err)
	}

	if cmd != nil {
		if err := applyChangedFlags(&cfg, cmd); err != nil {
			return nil, err
		}
	}

	if err := validate(&cfg); err != nil {
		return nil, err
	}

	if err := rejectPermissionAuto(cmd, projectRoot, &cfg); err != nil {
		return nil, err
	}

	cfg.ProjectRoot = projectRoot
	cfg.ProjectID = ProjectID(projectRoot)
	if opts.SkipProjectDataDir {
		cfg.ProjectDataDir, err = ProjectDataDir(projectRoot)
		if err != nil {
			return nil, err
		}
	} else {
		cfg.ProjectDataDir, err = EnsureProjectDataDir(projectRoot)
		if err != nil {
			return nil, err
		}
	}

	if opts.RequireAPIKey {
		cfg.APIKey, err = LoadAPIKey()
		if err != nil {
			return nil, err
		}
	}

	return &cfg, nil
}

func readYAMLIfExists(v *viper.Viper, pathFn func() (string, error)) error {
	path, err := pathFn()
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	v.SetConfigFile(path)
	if err := v.MergeInConfig(); err != nil {
		return fmt.Errorf("config: read %s: %w", path, err)
	}
	return nil
}

// BindFlags registers persistent CLI flags on the root command.
func BindFlags(cmd *cobra.Command) {
	fs := cmd.PersistentFlags()
	fs.String("model", "", "LLM model (deepseek-v4-pro|deepseek-v4-flash)")
	fs.String("subagent-model", "", "Subagent LLM model (deepseek-v4-pro|deepseek-v4-flash)")
	fs.Int("max-tokens", 0, "Max completion tokens (≤393216)")
	fs.String("thinking", "", "Thinking mode: enabled|disabled")
	fs.String("subagent-thinking", "", "Subagent thinking mode: enabled|disabled")
	fs.String("reasoning-effort", "", "Reasoning effort: high|max")
	fs.String("subagent-reasoning-effort", "", "Subagent reasoning effort: high|max")
	fs.Bool("strict-tools", false, "Use DeepSeek beta API for strict tool schema")
	fs.String("permission-mode", "", "Permission mode: readonly|ask|auto")
	fs.Bool("dangerously-auto", false, "Set permission mode to auto (use with care)")
	fs.Bool("plan", false, "Plan mode: read-only tools")
	fs.Bool("audit-log", false, "Enable audit log to project audit.jsonl")
	fs.StringP("prompt", "p", "", "Non-interactive single prompt")
	fs.Bool("json", false, "JSON output (with -p)")
	fs.CountP("verbose", "v", "Log level in project log file (-v INFO, -vv DEBUG)")
	fs.Bool("allow-log-sensitive-data", false, "Log full sensitive debug data (requires -vv)")
	fs.Bool("trace", false, "Enable OpenTelemetry spans and trace_id/span_id in logs")
	fs.String("trace-exporter", "", "Span exporter when tracing on: log|otlp")
	fs.String("trace-otlp-endpoint", "", "OTLP HTTP endpoint when --trace-exporter=otlp")
}
