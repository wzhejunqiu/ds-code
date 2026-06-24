package glob

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/tool"
	"github.com/wzhejunqiu/ds-code/internal/tool/builtin"
	"github.com/wzhejunqiu/ds-code/internal/tool/searchskip"
)

// GlobTool finds paths matching a glob under the workspace via ripgrep --files.
type GlobTool struct {
	Cfg        *config.Config
	Perm       *permission.Engine
	SearchSkip *searchskip.Matcher
	Strict     bool
}

func (t *GlobTool) Name() string { return tool.NameGlob.String() }

func (t *GlobTool) WithPerm(perm *permission.Engine) tool.Tool {
	cp := *t
	cp.Perm = perm
	return &cp
}

func (t *GlobTool) IsReadOnly() bool        { return true }
func (t *GlobTool) IsConcurrencySafe() bool { return true }

func (t *GlobTool) Description() string { return RenderDesc() }

func (t *GlobTool) Schema() map[string]any {
	return tool.ObjectSchema(map[string]any{
		"pattern": map[string]any{"type": "string", "description": SchemaPattern},
		"path":    map[string]any{"type": "string", "description": SchemaPath},
	}, []string{"pattern"}, t.Strict)
}

func (t *GlobTool) PermissionLevel() permission.Level { return permission.LevelLow }

func (t *GlobTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	in, err := parseGlobInput(args)
	if err != nil {
		return "", err
	}
	if in.Pattern == "" {
		return "", fmt.Errorf("%s", builtin.ErrPatternRequired)
	}
	return runRipgrepFiles(ctx, t, in)
}

var _ tool.Tool = (*GlobTool)(nil)
