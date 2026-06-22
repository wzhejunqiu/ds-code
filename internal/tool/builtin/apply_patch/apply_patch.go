package apply_patch

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/patch"
	patchapply "github.com/wzhejunqiu/ds-code/internal/patch/apply"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/tool"
	"github.com/wzhejunqiu/ds-code/internal/tool/builtin"
	"github.com/wzhejunqiu/ds-code/internal/tool/readgate"
)

// ApplyPatchTool applies Codex-style patch documents.
type ApplyPatchTool struct {
	Cfg    *config.Config
	Perm   *permission.Engine
	Strict bool
}

func (t *ApplyPatchTool) Name() string { return tool.NameApplyPatch.String() }

func (t *ApplyPatchTool) Description() string { return RenderDesc() }

func (t *ApplyPatchTool) Schema() map[string]any {
	return tool.ObjectSchema(map[string]any{
		"patch": map[string]any{
			"type":        "string",
			"description": SchemaPatchBody,
		},
	}, []string{"patch"}, t.Strict)
}

func (t *ApplyPatchTool) PermissionLevel() permission.Level { return permission.LevelHigh }

func (t *ApplyPatchTool) WithPerm(perm *permission.Engine) tool.Tool {
	cp := *t
	cp.Perm = perm
	return &cp
}

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
	validate := func(rel string) error {
		_, err := t.Perm.CheckWritablePath(rel)
		return err
	}
	required, err := patch.RequiredReadPaths(in.Patch, validate)
	if err != nil {
		return "", err
	}
	if gate, ok := readgate.FromContext(ctx); ok {
		if err := gate.CheckApplyPatch(required, ErrSameBatchReadEditFmt, ErrMustReadFirstFmt); err != nil {
			return "", err
		}
	}
	maxLines := t.Cfg.Tools.ApplyPatch.MaxChangedLines
	summary, err := patchapply.Apply(t.Perm.Workspace, in.Patch, t.Perm.CheckWritablePath, patchapply.Options{
		MaxChangedLines: maxLines,
	})
	if err != nil {
		return "", err
	}
	return ResultAppliedPrefix + summary, nil
}

var _ tool.Tool = (*ApplyPatchTool)(nil)
