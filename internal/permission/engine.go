package permission

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/wzhejunqiu/ds-code/internal/logging"
	"github.com/wzhejunqiu/ds-code/internal/patch"
	wspkg "github.com/wzhejunqiu/ds-code/internal/workspace"
	"go.uber.org/zap"
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
	writeTool   func(string) bool
}

// SetWriteToolDetector registers extra write tools (e.g. MCP mcp__* tools).
func (e *Engine) SetWriteToolDetector(fn func(string) bool) {
	e.writeTool = fn
}

// NewEngine creates a permission engine.
func NewEngine(mode, workspace string, interactive bool) *Engine {
	return &Engine{Mode: mode, Workspace: workspace, Interactive: interactive}
}

// Check validates whether a tool may run with the given arguments map.
func (e *Engine) Check(tool string, args map[string]any) error {
	err := e.check(tool, args)
	if err != nil {
		fields := []zap.Field{
			zap.String("tool", tool),
			zap.String("deny_reason", classifyDeny(err)),
		}
		if logging.AllowSensitiveData() {
			fields = append(fields, zap.Error(err))
		}
		logging.L().Debug("permission denied", fields...)
	}
	return err
}

func (e *Engine) check(tool string, args map[string]any) error {
	if tool == "shell" && isShellReadOnlyOp(args) {
		return e.checkShellReadOnly(args)
	}
	if e.isWriteTool(tool) && e.Mode == "readonly" {
		return fmt.Errorf("%w: %s in readonly mode", ErrDenied, tool)
	}
	if e.isWriteTool(tool) && e.Mode == "ask" && !e.Interactive {
		return ErrNeedTTY
	}

	if tool == "apply_patch" {
		patchText, _ := args["patch"].(string)
		if patchText != "" {
			paths, err := patch.Paths(patchText, e.Workspace)
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
		ok, err := e.Prompter(tool, e.summarizeArgs(tool, args))
		if err != nil {
			return err
		}
		if !ok {
			return ErrRejected
		}
	}
	return nil
}

func (e *Engine) summarizeArgs(tool string, args map[string]any) string {
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
			paths, err := patch.Paths(p, e.Workspace)
			if err == nil {
				return "files: " + strings.Join(paths, ", ")
			}
			return "patch (unparsed)"
		}
	default:
		if strings.HasPrefix(tool, "mcp__") {
			b, err := json.Marshal(args)
			if err == nil {
				s := string(b)
				if len(s) > 200 {
					s = s[:200] + "..."
				}
				return s
			}
		}
	}
	return ""
}

func isShellReadOnlyOp(args map[string]any) bool {
	if list, _ := args["list_jobs"].(bool); list {
		return true
	}
	jobID, _ := args["job_id"].(string)
	if jobID == "" {
		return false
	}
	if cancel, _ := args["cancel"].(bool); cancel {
		return false
	}
	return true
}

func (e *Engine) checkShellReadOnly(args map[string]any) error {
	// Poll/list background jobs: allowed in readonly; still blocked in ask without TTY for consistency with read ops.
	if e.Mode == "ask" && !e.Interactive {
		return ErrNeedTTY
	}
	return nil
}

func (e *Engine) isWriteTool(tool string) bool {
	if e.writeTool != nil && e.writeTool(tool) {
		return true
	}
	switch tool {
	case "shell", "write_file", "apply_patch":
		return true
	default:
		return false
	}
}

func (e *Engine) checkPath(rel string) error {
	_, err := e.CheckReadablePath(rel)
	return err
}

// CheckReadablePath resolves rel under the workspace and denies sensitive paths (S3).
func (e *Engine) CheckReadablePath(rel string) (string, error) {
	abs, err := e.ResolvePath(rel)
	if err != nil {
		logReadablePathDenied(rel, "", "resolve", err)
		return "", err
	}
	if IsSensitiveAbs(abs) {
		logReadablePathDenied(rel, abs, "sensitive", nil)
		return "", fmt.Errorf("%w: sensitive path %s", ErrDenied, rel)
	}
	return abs, nil
}

// ResolvePath resolves a path under workspace and blocks escape (S2: symlinks evaluated).
// Relative paths are preferred; absolute paths are accepted when they resolve inside workspace
// (models often pass full paths under project_root).
func (e *Engine) ResolvePath(rel string) (string, error) {
	abs, err := wspkg.ResolveRel(e.Workspace, rel)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrDenied, err)
	}
	return abs, nil
}

func logReadablePathDenied(rel, resolved, reason string, err error) {
	fields := []zap.Field{
		zap.Bool("allowed", false),
		zap.String("deny_reason", reason),
	}
	if logging.AllowSensitiveData() {
		fields = append(fields, logging.FieldString("path", rel))
		if resolved != "" {
			fields = append(fields, logging.FieldString("resolved", resolved))
		}
		if err != nil {
			fields = append(fields, zap.Error(err))
		}
	}
	logging.L().Debug("readable path denied", fields...)
}

func (e *Engine) checkSensitiveShell(cmd string) error {
	if err := e.checkShellDenylistPaths(cmd); err != nil {
		return err
	}
	if err := matchHighRiskShell(normalizeShellCmd(cmd)); err != nil {
		return err
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
