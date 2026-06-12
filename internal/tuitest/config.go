//go:build tuitest

package tuitest

import (
	"time"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/runmode"
)

// HarnessConfig builds a Config for TUI integration tests.
func HarnessConfig(mockBaseURL, projectRoot string) (*config.Config, error) {
	key, err := EnsureHarnessAPIKey()
	if err != nil {
		return nil, err
	}
	dataDir, err := config.EnsureProjectDataDir(projectRoot)
	if err != nil {
		return nil, err
	}
	return &config.Config{
		APIKey:         key,
		ProjectRoot:    projectRoot,
		ProjectID:      config.ProjectID(projectRoot),
		ProjectDataDir: dataDir,
		RunMode:        runmode.Agent,
		LLM: config.LLMConfig{
			BaseURL:         mockBaseURL,
			Model:           "deepseek-v4-pro",
			MaxTokens:       4096,
			Timeout:         120 * time.Second,
			StrictTools:     false,
			Thinking:        config.ThinkingConfig{Type: "enabled"},
			ReasoningEffort: "max",
		},
		Context: config.ContextConfig{
			WindowTokens:          1_048_576,
			MaxOutputTokens:       393_216,
			CompactThresholdRatio: 0.80,
			KeepRecentTurns:       6,
			TruncateBy:            "chars",
			ToolResultMaxChars:    100_000,
		},
		Agent:      config.AgentConfig{MaxTurns: 25},
		Permission: config.PermissionConfig{Mode: "auto"},
		Tools: config.ToolsConfig{
			ParallelToolCalls: true,
			ReadFile:          config.ReadFileToolConfig{MaxLines: 2000},
			Grep:              config.GrepToolConfig{HeadLimit: 200},
			Glob:              config.GlobToolConfig{MaxResults: 100},
			Shell:             config.ShellToolConfig{Timeout: 120 * time.Second},
		},
		MCP:   config.MCPConfig{Servers: nil},
		Web:   config.WebConfig{FetchEnabled: false, SearchEnabled: false},
		LSP:   config.LSPConfig{Enabled: false},
		Audit: config.AuditConfig{Enabled: false},
	}, nil
}
