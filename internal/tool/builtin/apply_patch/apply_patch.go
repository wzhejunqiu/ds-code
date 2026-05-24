package apply_patch

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/wzhejunqiu/ds-code/internal/config"
	patchapply "github.com/wzhejunqiu/ds-code/internal/patch/apply"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/tool"
	"github.com/wzhejunqiu/ds-code/internal/tool/builtin"
)

// ApplyPatchTool applies Codex-style patch documents.
type ApplyPatchTool struct {
	Cfg    *config.Config
	Perm   *permission.Engine
	Strict bool
}

func (t *ApplyPatchTool) Name() string { return "apply_patch" }

func (t *ApplyPatchTool) Description() string { return DescApplyPatch }

func (t *ApplyPatchTool) Schema() map[string]any {
	return tool.ObjectSchema(map[string]any{
		"patch": map[string]any{
			"type":        "string",
			"description": builtin.SchemaPatchBody,
		},
	}, []string{"patch"}, t.Strict)
}

func (t *ApplyPatchTool) PermissionLevel() permission.Level { return permission.LevelHigh }

func (t *ApplyPatchTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	var in struct {
		Patch string `json:"patch"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", err
	}
	if in.Patch == "" {
		return "", fmt.Errorf("%s", builtin.ErrPatchRequired)
	}
	maxLines := t.Cfg.Tools.ApplyPatch.MaxChangedLines
	summary, err := patchapply.Apply(t.Perm.Workspace, in.Patch, t.Perm.ResolvePath, patchapply.Options{
		MaxChangedLines: maxLines,
	})
	if err != nil {
		return "", err
	}
	return ResultAppliedPrefix + summary, nil
}

var _ tool.Tool = (*ApplyPatchTool)(nil)
