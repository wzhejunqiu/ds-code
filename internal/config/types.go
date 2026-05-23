// Package config loads and validates ds-code settings (see README.md and docs/CONFIG.md).
package config

import (
	"regexp"
	"time"
)

// Config is the merged runtime configuration for ds-code.
// Fields tagged mapstructure:"-" are filled after YAML/CLI merge (paths, API key, CLI-only).
type Config struct {
	LLM            LLMConfig            `mapstructure:"llm"`
	Context          ContextConfig        `mapstructure:"context"`
	Agent            AgentConfig          `mapstructure:"agent"`
	Tools            ToolsConfig          `mapstructure:"tools"`
	Permission       PermissionConfig     `mapstructure:"permission"`
	BTW              BTWConfig            `mapstructure:"btw"`
	NonInteractive   NonInteractiveConfig `mapstructure:"non_interactive"`
	Audit            AuditConfig            `mapstructure:"audit"`
	MCP              MCPConfig            `mapstructure:"mcp"`
	Web              WebConfig            `mapstructure:"web"`
	LSP              LSPConfig            `mapstructure:"lsp"`
	RunMode          string               `mapstructure:"run_mode"`
	LogVerbosity          int  `mapstructure:"-"`
	AllowLogSensitiveData bool `mapstructure:"-"`
	JSONOutput            bool `mapstructure:"-"`
	Prompt           string               `mapstructure:"-"`
	ProjectRoot      string               `mapstructure:"-"`
	ProjectID        string               `mapstructure:"-"`
	ProjectDataDir   string               `mapstructure:"-"`
	APIKey           string               `mapstructure:"-"`
}

type LLMConfig struct {
	BaseURL         string        `mapstructure:"base_url"`
	Model           string        `mapstructure:"model"`
	MaxTokens       int           `mapstructure:"max_tokens"`
	Timeout         time.Duration `mapstructure:"timeout"`
	StrictTools     bool          `mapstructure:"strict_tools"`
	Thinking        ThinkingConfig `mapstructure:"thinking"`
	ReasoningEffort string        `mapstructure:"reasoning_effort"`
}

type ThinkingConfig struct {
	Type string `mapstructure:"type"`
}

type ContextConfig struct {
	WindowTokens           int     `mapstructure:"window_tokens"`
	MaxOutputTokens        int     `mapstructure:"max_output_tokens"`
	CompactThresholdRatio  float64 `mapstructure:"compact_threshold_ratio"`
	KeepRecentTurns        int     `mapstructure:"keep_recent_turns"`
	TruncateBy             string  `mapstructure:"truncate_by"`
	ToolResultMaxChars     int     `mapstructure:"tool_result_max_chars"`
	AtReferenceMaxChars    int     `mapstructure:"at_reference_max_chars"`
	GitSnapshotMaxChars    int     `mapstructure:"git_snapshot_max_chars"`
	AtDirMaxFiles          int     `mapstructure:"at_dir_max_files"`
	AtDirMaxDepth          int     `mapstructure:"at_dir_max_depth"`
}

type AgentConfig struct {
	MaxTurns int `mapstructure:"max_turns"`
}

type ToolsConfig struct {
	ParallelToolCalls bool              `mapstructure:"parallel_tool_calls"`
	ReadFile          ReadFileToolConfig `mapstructure:"read_file"`
	Grep              GrepToolConfig     `mapstructure:"grep"`
	Glob              GlobToolConfig     `mapstructure:"glob"`
	ApplyPatch        ApplyPatchConfig   `mapstructure:"apply_patch"`
	Shell             ShellToolConfig    `mapstructure:"shell"`
	Task              TaskToolConfig     `mapstructure:"task"`
}

type ReadFileToolConfig struct {
	MaxLines int `mapstructure:"max_lines"`
	MaxBytes int `mapstructure:"max_bytes"`
}

type GrepToolConfig struct {
	HeadLimit int `mapstructure:"head_limit"`
}

type GlobToolConfig struct {
	MaxResults int `mapstructure:"max_results"`
}

type ApplyPatchConfig struct {
	MaxChangedLines int `mapstructure:"max_changed_lines"`
}

type ShellToolConfig struct {
	Timeout                  time.Duration `mapstructure:"timeout"`
	MaxBackground            int           `mapstructure:"max_background"`
	BackgroundOutputMaxBytes int           `mapstructure:"background_output_max_bytes"`
	EnvBlacklist             []string      `mapstructure:"env_blacklist"`
	EnvBlacklistCompiled     []*regexp.Regexp `mapstructure:"-"`
}

type TaskToolConfig struct {
	MaxParallel      int `mapstructure:"max_parallel"`
	SummaryMaxChars  int `mapstructure:"summary_max_chars"`
}

type PermissionConfig struct {
	Mode string `mapstructure:"mode"`
}

type BTWConfig struct {
	IncludeRecentTurns int  `mapstructure:"include_recent_turns"`
	MaxTokens          int  `mapstructure:"max_tokens"`
	CountTowardSession bool `mapstructure:"count_toward_session"`
}

type NonInteractiveConfig struct {
	EphemeralSession bool `mapstructure:"ephemeral_session"`
}

type AuditConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

type MCPConfig struct {
	Servers []MCPServerConfig `mapstructure:"servers"`
}

type MCPServerConfig struct {
	Name    string            `mapstructure:"name"`
	Command string            `mapstructure:"command"`
	Args    []string          `mapstructure:"args"`
	Env     map[string]string `mapstructure:"env"`
}

type WebConfig struct {
	FetchEnabled  bool     `mapstructure:"fetch_enabled"`
	SearchEnabled bool     `mapstructure:"search_enabled"`
	Allowlist     []string `mapstructure:"allowlist"`
}

type LSPConfig struct {
	Enabled              bool                       `mapstructure:"enabled"`
	IdleShutdown         time.Duration              `mapstructure:"idle_shutdown"`
	DiagnosticsTimeout   time.Duration              `mapstructure:"diagnostics_timeout"`
	MaxFilesPerCall      int                        `mapstructure:"max_files_per_call"`
	MaxIssuesPerFile     int                        `mapstructure:"max_issues_per_file"`
	WarmupOnStart        []string                   `mapstructure:"warmup_on_start"`
	Servers              map[string]LSPServerConfig `mapstructure:"servers"`
}

type LSPServerConfig struct {
	Command     string            `mapstructure:"command"`
	Args        []string          `mapstructure:"args"`
	Extensions  []string          `mapstructure:"extensions"`
	Env         map[string]string `mapstructure:"env"`
	Disabled    bool              `mapstructure:"disabled"`
}
