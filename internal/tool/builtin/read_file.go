package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/permission"
	"github.com/hejunqiu/ds-code/internal/tool"
)

// ReadFileTool reads file contents with optional line range.
type ReadFileTool struct {
	Cfg    *config.Config
	Perm   *permission.Engine
	Strict bool
}

func (t *ReadFileTool) Name() string { return "read_file" }

func (t *ReadFileTool) Description() string {
	return "Read a file under the project workspace. Optional offset (1-based) and limit (max lines per call)."
}

func (t *ReadFileTool) Schema() map[string]any {
	return tool.ObjectSchema(map[string]any{
		"path":   map[string]any{"type": "string", "description": "Relative path from project root"},
		"offset": map[string]any{"type": "integer", "description": "Start line (1-based)"},
		"limit":  map[string]any{"type": "integer", "description": "Max lines to read"},
	}, []string{"path"}, t.Strict)
}

func (t *ReadFileTool) PermissionLevel() permission.Level { return permission.LevelLow }

func (t *ReadFileTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	var in struct {
		Path   string `json:"path"`
		Offset int    `json:"offset"`
		Limit  int    `json:"limit"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", err
	}
	if in.Path == "" {
		return "", fmt.Errorf("path is required")
	}
	abs, err := t.Perm.ResolvePath(in.Path)
	if err != nil {
		return "", err
	}
	limit := in.Limit
	if limit <= 0 {
		limit = t.Cfg.Tools.ReadFile.MaxLines
	}
	if limit > t.Cfg.Tools.ReadFile.MaxLines {
		limit = t.Cfg.Tools.ReadFile.MaxLines
	}

	b, err := os.ReadFile(abs)
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(b), "\n")
	start := 0
	if in.Offset > 0 {
		start = in.Offset - 1
	}
	if start >= len(lines) {
		return fmt.Sprintf("(empty: offset %d beyond file length %d)", in.Offset, len(lines)), nil
	}
	end := start + limit
	if end > len(lines) {
		end = len(lines)
	}
	var out strings.Builder
	for i := start; i < end; i++ {
		fmt.Fprintf(&out, "%d|%s\n", i+1, lines[i])
	}
	if end < len(lines) {
		fmt.Fprintf(&out, "\n... %d more lines not shown", len(lines)-end)
	}
	return out.String(), nil
}

var _ tool.Tool = (*ReadFileTool)(nil)
