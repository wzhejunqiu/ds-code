package shell

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/tool"
)

const maxShellTimeoutMs = 600_000

// ResolveShellTimeout returns the effective timeout for a bash command (sync or background).
func ResolveShellTimeout(cfg *config.Config, timeoutMs int) (time.Duration, error) {
	if timeoutMs < 0 {
		return 0, fmt.Errorf("%s", ErrTimeoutMsNonNegative)
	}
	if timeoutMs > 0 {
		ms := timeoutMs
		if ms > maxShellTimeoutMs {
			ms = maxShellTimeoutMs
		}
		return time.Duration(ms) * time.Millisecond, nil
	}
	timeout := cfg.Tools.Shell.Timeout
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	return timeout, nil
}

// ResolveSyncTimeout is an alias for ResolveShellTimeout.
func ResolveSyncTimeout(cfg *config.Config, timeoutMs int) (time.Duration, error) {
	return ResolveShellTimeout(cfg, timeoutMs)
}

// ShellTimeoutDeadline returns the TUI countdown deadline for bash commands.
func ShellTimeoutDeadline(now time.Time, cfg *config.Config, rawArgs []byte) (time.Time, bool) {
	return shellTimeoutDeadline(now, cfg, rawArgs)
}

// ShellSyncTimeoutDeadline is an alias for ShellTimeoutDeadline.
func ShellSyncTimeoutDeadline(now time.Time, cfg *config.Config, rawArgs []byte) (time.Time, bool) {
	return shellTimeoutDeadline(now, cfg, rawArgs)
}

func shellTimeoutDeadline(now time.Time, cfg *config.Config, rawArgs []byte) (time.Time, bool) {
	if cfg == nil {
		return time.Time{}, false
	}
	args := tool.ArgsMap(rawArgs)
	cmd, _ := args["command"].(string)
	if strings.TrimSpace(cmd) == "" {
		return time.Time{}, false
	}
	timeoutMs := timeoutMsFromArgs(args)
	d, err := ResolveShellTimeout(cfg, timeoutMs)
	if err != nil {
		return time.Time{}, false
	}
	return now.Add(d), true
}

func timeoutMsFromArgs(args map[string]any) int {
	switch v := args["timeout_ms"].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	case json.Number:
		i, _ := v.Int64()
		return int(i)
	default:
		return 0
	}
}
