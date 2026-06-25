package write_file

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/tool"
	"github.com/wzhejunqiu/ds-code/internal/tool/builtin"
	"github.com/wzhejunqiu/ds-code/internal/tool/readgate"
)

// WriteFileTool creates or overwrites a whole file.
type WriteFileTool struct {
	Cfg    *config.Config
	Perm   *permission.Engine
	Strict bool
}

func (t *WriteFileTool) Name() string { return tool.NameWriteFile.String() }

func (t *WriteFileTool) WithPerm(perm *permission.Engine) tool.Tool {
	cp := *t
	cp.Perm = perm
	return &cp
}

func (t *WriteFileTool) Description() string { return RenderDesc() }

func (t *WriteFileTool) Schema() map[string]any {
	return tool.ObjectSchema(map[string]any{
		"path": map[string]any{
			"type":        "string",
			"description": builtin.SchemaPathRelRoot,
		},
		"content": map[string]any{
			"type":        "string",
			"description": builtin.SchemaFullFileContent,
		},
	}, []string{"path", "content"}, t.Strict)
}

func (t *WriteFileTool) PermissionLevel() permission.Level { return permission.LevelHigh }

func (t *WriteFileTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	var in struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", err
	}
	if in.Path == "" {
		return "", fmt.Errorf("%s", builtin.ErrPathRequired)
	}
	abs, err := t.Perm.ResolvePath(in.Path)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(abs); err == nil {
		if gate, ok := readgate.FromContext(ctx); ok {
			if err := gate.CheckApplyPatch(
				[]string{in.Path},
				ErrSameBatchReadWriteFmt,
				ErrMustReadFirstFmt,
			); err != nil {
				return "", err
			}
		}
	} else if !os.IsNotExist(err) {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(abs, []byte(in.Content), 0o644); err != nil {
		return "", err
	}
	return fmt.Sprintf(ResultWrote, in.Path, len(in.Content)), nil
}

var _ tool.Tool = (*WriteFileTool)(nil)
