package agent

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/datadir"
	"github.com/wzhejunqiu/ds-code/internal/logging"
	"go.uber.org/zap"
)

// HookEvent names the lifecycle point where a hook fires.
type HookEvent string

const (
	HookPreToolUse    HookEvent = "PreToolUse"
	HookPostToolUse   HookEvent = "PostToolUse"
	HookStop          HookEvent = "Stop"
	HookSubagentStart HookEvent = "SubagentStart"
	HookSubagentStop  HookEvent = "SubagentStop"
	HookSessionStart  HookEvent = "SessionStart"
	HookSessionEnd    HookEvent = "SessionEnd"
)

// HookConfig is a single hook definition from .ds-code/hooks.json.
type HookConfig struct {
	Event   HookEvent `json:"event"`
	Command string    `json:"command"`
	Timeout string    `json:"timeout,omitempty"`
}

// HookResult captures the output of a hook execution.
type HookResult struct {
	Event  HookEvent
	Output string
	Error  error
}

// HookManager loads and executes hooks from .ds-code/hooks.json.
type HookManager struct {
	configs []HookConfig
}

// LoadHooks reads hooks.json from the project root if present.
func LoadHooks(projectRoot string) *HookManager {
	path := filepath.Join(datadir.ProjectMetadataDir(projectRoot), "hooks.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return &HookManager{}
	}
	var configs []HookConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		logging.L().Warn("failed to parse hooks.json", zap.String("path", path), zap.Error(err))
		return &HookManager{}
	}
	logging.L().Info("hooks loaded", zap.String("path", path), zap.Int("count", len(configs)))
	return &HookManager{configs: configs}
}

// Run executes all hooks registered for the given event.
func (hm *HookManager) Run(ctx context.Context, event HookEvent, input string) []HookResult {
	var results []HookResult
	for _, c := range hm.configs {
		if c.Event != event {
			continue
		}
		results = append(results, hm.runOne(ctx, c, event, input))
	}
	return results
}

func (hm *HookManager) runOne(ctx context.Context, c HookConfig, event HookEvent, input string) HookResult {
	timeout := 30 * time.Second
	if c.Timeout != "" {
		if d, err := time.ParseDuration(c.Timeout); err == nil {
			timeout = d
		}
	}
	hookCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(hookCtx, "sh", "-c", c.Command)
	cmd.Env = os.Environ()
	if input != "" {
		cmd.Env = append(cmd.Env, "HOOK_INPUT="+input)
	}
	output, err := cmd.CombinedOutput()
	result := HookResult{Event: event, Output: string(output)}
	if err != nil {
		result.Error = err
		logging.L().Debug("hook failed",
			zap.String("event", string(event)),
			zap.String("command", c.Command),
			zap.Error(err),
		)
	}
	return result
}
