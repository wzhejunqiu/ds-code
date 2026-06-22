package permission

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/wzhejunqiu/ds-code/internal/logging"
	"github.com/wzhejunqiu/ds-code/internal/patch"
	"github.com/wzhejunqiu/ds-code/internal/security/classifier"
	"github.com/wzhejunqiu/ds-code/internal/toolname"
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
	ProjectRoot string // cfg.ProjectRoot for spill paths (may differ from Workspace in worktrees)
	// SpillSessionID is set per RunTurn for logging/trace; not used to bind spill read_file access.
	SpillSessionID string
	Interactive    bool
	Prompter       Prompter
	writeTool      func(string) bool
	mcpTool        func(string) bool
}

// SetWriteToolDetector registers extra write tools (e.g. MCP write tools).
func (e *Engine) SetWriteToolDetector(fn func(string) bool) {
	e.writeTool = fn
}

// SetMCPToolDetector registers MCP bare tool names for ask-mode arg summaries.
func (e *Engine) SetMCPToolDetector(fn func(string) bool) {
	e.mcpTool = fn
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
	if tool == toolname.Bash {
		if cmd, _ := args["command"].(string); cmd != "" {
			handled, err := e.checkShellCommand(cmd)
			if err != nil {
				return err
			}
			if handled {
				return nil
			}
		}
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
			paths, err := patch.Paths(patchText, e.patchValidator())
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

	if path := pathArgForCheck(tool, args); path != "" {
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
	if tool == toolname.Bash {
		if cmd, _ := args["command"].(string); cmd != "" {
			if err := e.checkSensitiveShell(cmd); err != nil {
				return err
			}
		}
	}

	if e.isWriteTool(tool) && e.Mode == "ask" && e.Interactive {
		if tool == toolname.Bash {
			if cmd, _ := args["command"].(string); cmd != "" {
				if dec, _ := classifier.Classify(cmd); dec == classifier.Ask || dec == classifier.Allow {
					return nil
				}
			}
		}
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

// checkShellCommand applies classifier rules for shell commands.
// When handled is true, the caller should return immediately (no further write-tool checks).
func (e *Engine) checkShellCommand(cmd string) (handled bool, err error) {
	dec, reason := classifier.Classify(cmd)
	switch dec {
	case classifier.Deny:
		return true, fmt.Errorf("%w: %s", ErrDenied, reason)
	case classifier.Allow:
		if err := e.checkSensitiveShell(cmd); err != nil {
			return true, err
		}
		if e.Mode == "readonly" {
			return true, nil
		}
		if e.Mode == "ask" && e.Interactive {
			return true, nil
		}
		return false, nil
	case classifier.Ask:
		if e.Mode == "readonly" {
			return true, fmt.Errorf("%w: %s", ErrDenied, reason)
		}
		if err := e.checkSensitiveShell(cmd); err != nil {
			return true, err
		}
		if !e.Interactive || e.Prompter == nil {
			return true, fmt.Errorf("%w: %s requires approval (non-interactive)", ErrDenied, reason)
		}
		ok, err := e.Prompter(toolname.Bash, cmd)
		if err != nil {
			return true, err
		}
		if !ok {
			return true, ErrRejected
		}
		return true, nil
	default:
		return false, nil
	}
}

func (e *Engine) summarizeArgs(tool string, args map[string]any) string {
	switch tool {
	case toolname.Bash:
		if c, _ := args["command"].(string); c != "" {
			return c
		}
	case "write_file":
		if p, _ := args["path"].(string); p != "" {
			return "path=" + p
		}
	case "apply_patch":
		if p, _ := args["patch"].(string); p != "" {
			paths, err := patch.Paths(p, e.patchValidator())
			if err == nil {
				return "files: " + strings.Join(paths, ", ")
			}
			return "patch (unparsed)"
		}
	default:
		if e.isMCPToolName(tool) || strings.HasPrefix(tool, "mcp__") {
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

func (e *Engine) isMCPToolName(tool string) bool {
	if e.mcpTool != nil && e.mcpTool(tool) {
		return true
	}
	return false
}

func (e *Engine) isWriteTool(tool string) bool {
	if e.writeTool != nil && e.writeTool(tool) {
		return true
	}
	switch tool {
	case toolname.Bash, "write_file", "apply_patch":
		return true
	default:
		return false
	}
}

// PatchValidator returns a path validator for patch parse (S2 boundary).
func (e *Engine) PatchValidator() patch.PathValidator {
	return e.patchValidator()
}

func (e *Engine) patchValidator() patch.PathValidator {
	if e.Workspace == "" {
		return nil
	}
	return func(rel string) error {
		_, err := e.ResolveAccessPath(rel, PathBoundary)
		return err
	}
}

func (e *Engine) checkPath(rel string) error {
	_, err := e.CheckReadablePath(rel)
	return err
}

// CheckReadablePath resolves rel under the workspace and denies sensitive paths (S3).
// Regular files under the current project data directory are allowed (see resolveProjectDataRead).
func (e *Engine) CheckReadablePath(rel string) (string, error) {
	if abs, ok := e.resolveProjectDataRead(rel); ok {
		return abs, nil
	}
	return e.ResolveAccessPath(rel, PathRead)
}

func pathArgForCheck(tool string, args map[string]any) string {
	if tool == "read_file" {
		if fp, _ := args["filepath"].(string); fp != "" {
			return fp
		}
		return ""
	}
	if p, _ := args["path"].(string); p != "" {
		return p
	}
	return ""
}

// ResolvePath resolves a path under workspace and blocks escape (S2: symlinks evaluated).
// Relative paths are preferred; absolute paths are accepted when they resolve inside workspace
// (models often pass full paths under project_root).
func (e *Engine) ResolvePath(rel string) (string, error) {
	abs, err := wspkg.ResolveRel(e.Workspace, rel)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrDenied, err)
	}
	return abs, nil
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
