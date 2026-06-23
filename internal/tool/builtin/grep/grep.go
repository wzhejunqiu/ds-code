package grep

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

// GrepTool searches file contents via ripgrep.
type GrepTool struct {
	Cfg        *config.Config
	Perm       *permission.Engine
	SearchSkip *searchskip.Matcher
	Strict     bool
}

type grepInput struct {
	Pattern    string `json:"pattern"`
	Path       string `json:"path"`
	Glob       string `json:"glob"`
	OutputMode string `json:"output_mode"`
	Before     int    `json:"-B"`
	After      int    `json:"-A"`
	ContextC   int    `json:"-C"`
	Context    int    `json:"context"`
	LineNums   *bool  `json:"-n"`
	IgnoreCase bool   `json:"-i"`
	Type       string `json:"type"`
	HeadLimit  *int   `json:"head_limit"`
	Offset     int    `json:"offset"`
	Multiline  bool   `json:"multiline"`
}

func (t *GrepTool) Name() string { return tool.NameGrep.String() }

func (t *GrepTool) WithPerm(perm *permission.Engine) tool.Tool {
	cp := *t
	cp.Perm = perm
	return &cp
}

func (t *GrepTool) IsReadOnly() bool        { return true }
func (t *GrepTool) IsConcurrencySafe() bool { return true }

func (t *GrepTool) Description() string { return RenderDesc() }

func (t *GrepTool) Schema() map[string]any {
	return tool.ObjectSchema(map[string]any{
		"pattern": map[string]any{
			"type":        "string",
			"description": SchemaPattern,
		},
		"path": map[string]any{
			"type":        "string",
			"description": SchemaPath,
		},
		"glob": map[string]any{
			"type":        "string",
			"description": SchemaGlob,
		},
		"output_mode": map[string]any{
			"type": "string",
			"enum": []string{
				builtin.GrepOutputContent,
				builtin.GrepOutputFilesWithMatches,
				builtin.GrepOutputCount,
			},
			"description": SchemaOutputMode,
		},
		"-B": map[string]any{
			"type":        "number",
			"description": SchemaBefore,
		},
		"-A": map[string]any{
			"type":        "number",
			"description": SchemaAfter,
		},
		"-C": map[string]any{
			"type":        "number",
			"description": SchemaContextC,
		},
		"context": map[string]any{
			"type":        "number",
			"description": SchemaContext,
		},
		"-n": map[string]any{
			"type":        "boolean",
			"description": SchemaLineNumbers,
		},
		"-i": map[string]any{
			"type":        "boolean",
			"description": SchemaIgnoreCase,
		},
		"type": map[string]any{
			"type":        "string",
			"description": SchemaFileType,
		},
		"head_limit": map[string]any{
			"type":        "number",
			"description": SchemaHeadLimit,
		},
		"offset": map[string]any{
			"type":        "number",
			"description": SchemaOffset,
		},
		"multiline": map[string]any{
			"type":        "boolean",
			"description": SchemaMultiline,
		},
	}, []string{"pattern"}, t.Strict)
}

func (t *GrepTool) PermissionLevel() permission.Level { return permission.LevelLow }

func (t *GrepTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	var in grepInput
	if err := json.Unmarshal(args, &in); err != nil {
		return "", err
	}
	if in.Pattern == "" {
		return "", fmt.Errorf("%s", builtin.ErrPatternRequired)
	}
	if len(in.Pattern) > 512 {
		return "", fmt.Errorf("%s", builtin.ErrPatternTooLong)
	}
	if _, err := builtin.ParseGrepOutputMode(in.OutputMode); err != nil {
		return "", err
	}
	return runRipgrep(ctx, t, in)
}

var _ tool.Tool = (*GrepTool)(nil)
