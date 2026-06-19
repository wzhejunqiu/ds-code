package permission

import (
	"errors"
	"fmt"
	"path/filepath"

	wspkg "github.com/wzhejunqiu/ds-code/internal/workspace"
)

// PathIntent distinguishes read/write/boundary/enumerate path checks.
type PathIntent int

const (
	PathRead PathIntent = iota
	PathWrite
	PathBoundary
	PathEnumerate
)

// SkipSensitiveAbs reports whether abs is on the sensitive denylist (S3) and should be skipped during enumeration.
func (e *Engine) SkipSensitiveAbs(abs string) bool {
	return IsSensitiveAbs(abs)
}

// ResolveAccessPath resolves rel under the workspace and applies S2+S3 per intent.
func (e *Engine) ResolveAccessPath(rel string, intent PathIntent) (string, error) {
	abs, err := wspkg.ResolveRel(e.Workspace, rel)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrDenied, err)
	}
	if intent == PathRead || intent == PathWrite {
		if IsSensitiveAbs(abs) {
			return "", fmt.Errorf("%w: sensitive path %s", ErrDenied, rel)
		}
	}
	return abs, nil
}

// CheckWritablePath resolves rel for write operations (S2 + S3).
func (e *Engine) CheckWritablePath(rel string) (string, error) {
	return e.ResolveAccessPath(rel, PathWrite)
}

// CheckAbsPath validates an absolute path against workspace boundary and optional S3.
func (e *Engine) CheckAbsPath(abs string, intent PathIntent) error {
	abs = filepath.Clean(abs)
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		abs = resolved
	}
	if err := wspkg.EnsureAbsUnder(e.Workspace, abs); err != nil {
		return fmt.Errorf("%w: outside workspace: %s", ErrDenied, abs)
	}
	if intent == PathRead || intent == PathWrite {
		if IsSensitiveAbs(abs) {
			return fmt.Errorf("%w: sensitive path %s", ErrDenied, abs)
		}
	}
	return nil
}

func isOutsideWorkspaceErr(err error) bool {
	return errors.Is(err, wspkg.ErrOutsideWorkspace)
}
