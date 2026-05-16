package permission

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hejunqiu/ds-code/internal/patch"
)

// ErrDenied is returned when an operation is not allowed.
var ErrDenied = errors.New("permission denied")

// ErrNeedTTY is returned when ask mode cannot prompt in non-interactive mode.
var ErrNeedTTY = errors.New("permission: ask mode requires a TTY; use --permission-mode readonly or --dangerously-auto")

// ErrRejected is returned when the user declines a prompt.
var ErrRejected = errors.New("permission: user rejected")

// Engine enforces permission mode and path policies.
type Engine struct {
	Mode        string
	Workspace   string
	Interactive bool
	Prompter    Prompter
}

// NewEngine creates a permission engine.
func NewEngine(mode, workspace string, interactive bool) *Engine {
	return &Engine{Mode: mode, Workspace: workspace, Interactive: interactive}
}

// Check validates whether a tool may run with the given arguments map.
func (e *Engine) Check(tool string, args map[string]any) error {
	if e.isWriteTool(tool) && e.Mode == "readonly" {
		return fmt.Errorf("%w: %s in readonly mode", ErrDenied, tool)
	}
	if e.isWriteTool(tool) && e.Mode == "ask" && !e.Interactive {
		return ErrNeedTTY
	}

	if tool == "apply_patch" {
		patchText, _ := args["patch"].(string)
		if patchText != "" {
			paths, err := patch.Paths(patchText)
			if err != nil {
				return fmt.Errorf("%w: invalid patch: %v", ErrDenied, err)
			}
			for _, p := range paths {
				if err := e.checkPath(p); err != nil {
					return err
				}
			}
		}
	}

	if path, _ := args["path"].(string); path != "" {
		if err := e.checkPath(path); err != nil {
			return err
		}
	}
	if paths, ok := args["paths"].([]any); ok {
		for _, p := range paths {
			if ps, ok := p.(string); ok {
				if err := e.checkPath(ps); err != nil {
					return err
				}
			}
		}
	}
	if tool == "shell" {
		if cmd, _ := args["command"].(string); cmd != "" {
			if err := e.checkSensitiveShell(cmd); err != nil {
				return err
			}
		}
	}

	if e.isWriteTool(tool) && e.Mode == "ask" && e.Interactive {
		if e.Prompter == nil {
			return fmt.Errorf("%w: no prompter configured for ask mode", ErrDenied)
		}
		ok, err := e.Prompter(tool, summarizeArgs(tool, args))
		if err != nil {
			return err
		}
		if !ok {
			return ErrRejected
		}
	}
	return nil
}

func summarizeArgs(tool string, args map[string]any) string {
	switch tool {
	case "shell":
		if c, _ := args["command"].(string); c != "" {
			return c
		}
	case "write_file":
		if p, _ := args["path"].(string); p != "" {
			return "path=" + p
		}
	case "apply_patch":
		if p, _ := args["patch"].(string); p != "" {
			paths, err := patch.Paths(p)
			if err == nil {
				return "files: " + strings.Join(paths, ", ")
			}
			return "patch (unparsed)"
		}
	}
	return ""
}

func (e *Engine) isWriteTool(tool string) bool {
	switch tool {
	case "shell", "write_file", "apply_patch":
		return true
	default:
		return false
	}
}

func (e *Engine) checkPath(rel string) error {
	abs, err := e.ResolvePath(rel)
	if err != nil {
		return err
	}
	if e.isSensitive(abs) {
		return fmt.Errorf("%w: sensitive path %s", ErrDenied, rel)
	}
	return nil
}

// ResolvePath resolves rel under workspace and blocks escape (S2: symlinks evaluated).
func (e *Engine) ResolvePath(rel string) (string, error) {
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("%w: absolute paths not allowed: %s", ErrDenied, rel)
	}
	if strings.Contains(rel, "..") {
		return "", fmt.Errorf("%w: path traversal: %s", ErrDenied, rel)
	}

	ws, err := filepath.EvalSymlinks(e.Workspace)
	if err != nil {
		ws, err = filepath.Abs(e.Workspace)
		if err != nil {
			return "", err
		}
	}

	abs := filepath.Join(ws, filepath.Clean(rel))
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		abs = resolved
	} else if _, statErr := os.Stat(abs); statErr != nil {
		// New file: resolve parent directory
		parent, err := filepath.EvalSymlinks(filepath.Dir(abs))
		if err != nil {
			return "", fmt.Errorf("%w: cannot resolve parent of %s", ErrDenied, rel)
		}
		abs = filepath.Join(parent, filepath.Base(abs))
	}

	relTo, err := filepath.Rel(ws, abs)
	if err != nil || strings.HasPrefix(relTo, "..") {
		return "", fmt.Errorf("%w: outside workspace: %s", ErrDenied, rel)
	}
	return abs, nil
}

var sensitivePatterns = []string{
	".env",
	".ssh",
	"id_rsa",
	"id_ed25519",
	"credentials",
	"secrets",
}

func (e *Engine) isSensitive(abs string) bool {
	lower := strings.ToLower(filepath.ToSlash(abs))
	for _, p := range sensitivePatterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

func (e *Engine) checkSensitiveShell(cmd string) error {
	lower := strings.ToLower(cmd)
	for _, p := range []string{"rm -rf /", "mkfs", ":(){", "curl | sh", "wget | sh"} {
		if strings.Contains(lower, p) {
			return fmt.Errorf("%w: high-risk shell command blocked", ErrDenied)
		}
	}
	return nil
}

// IsInteractiveTTY reports whether stdin is a terminal.
func IsInteractiveTTY() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
