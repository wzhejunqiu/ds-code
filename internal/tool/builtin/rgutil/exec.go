package rgutil

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/tool/builtin"
	"github.com/wzhejunqiu/ds-code/internal/tool/builtin/grep/rgbin"
)

// ErrTimeout is returned when the ripgrep subprocess exceeds the configured timeout.
var ErrTimeout = errors.New("ripgrep: search timed out")

// ResolveBinary returns the ripgrep executable path from tools.grep binary settings.
func ResolveBinary(cfg config.GrepToolConfig) (string, error) {
	switch cfg.Binary {
	case "", "bundled":
		return rgbin.Path()
	case "system":
		path, err := exec.LookPath("rg")
		if err != nil {
			return "", fmt.Errorf("ripgrep: rg not found in PATH")
		}
		return path, nil
	case "path":
		if cfg.BinaryPath == "" {
			return "", fmt.Errorf("tools.grep.binary_path required when binary=path")
		}
		return validateExecutable(cfg.BinaryPath)
	default:
		return "", fmt.Errorf("unknown tools.grep.binary: %q", cfg.Binary)
	}
}

func validateExecutable(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("ripgrep: binary_path: %w", err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("ripgrep: binary_path is a directory")
	}
	if info.Mode()&0o111 == 0 {
		return "", fmt.Errorf("ripgrep: binary_path is not executable")
	}
	return path, nil
}

// Exec runs ripgrep with args in workspace. Exit code 1 (no matches) is success.
func Exec(ctx context.Context, rgPath string, args []string, workspace string, timeout time.Duration) (stdout, stderr string, err error) {
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, rgPath, args...)
	cmd.Dir = workspace
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	runErr := cmd.Run()
	if runCtx.Err() == context.DeadlineExceeded {
		return outBuf.String(), errBuf.String(), ErrTimeout
	}
	if errors.Is(ctx.Err(), context.Canceled) {
		return "", "", ctx.Err()
	}
	if runErr != nil {
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			if exitErr.ExitCode() == 1 {
				return outBuf.String(), errBuf.String(), nil
			}
			return outBuf.String(), errBuf.String(), MapStderr(errBuf.String(), exitErr.ExitCode())
		}
		return outBuf.String(), errBuf.String(), runErr
	}
	return outBuf.String(), errBuf.String(), nil
}

// MapStderr maps ripgrep stderr to a user-facing error.
func MapStderr(stderr string, code int) error {
	stderr = bytesTrimSpace(stderr)
	if containsRegexParseError(stderr) {
		if stderr != "" {
			return fmt.Errorf("%s: %s", builtin.ErrInvalidRegex, stderr)
		}
		return fmt.Errorf("%s", builtin.ErrInvalidRegex)
	}
	if stderr != "" {
		return fmt.Errorf("ripgrep exited with code %d: %s", code, stderr)
	}
	return fmt.Errorf("ripgrep exited with code %d", code)
}

func bytesTrimSpace(s string) string {
	return strings.TrimSpace(s)
}

func containsRegexParseError(stderr string) bool {
	return strings.Contains(stderr, "regex parse error") || strings.Contains(stderr, "error parsing regex")
}
