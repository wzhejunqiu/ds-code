package builtin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/patch"
	"github.com/hejunqiu/ds-code/internal/permission"
	"github.com/hejunqiu/ds-code/internal/tool"
)

// ApplyPatchTool applies Codex-style patch documents.
type ApplyPatchTool struct {
	Cfg    *config.Config
	Perm   *permission.Engine
	Strict bool
}

func (t *ApplyPatchTool) Name() string { return "apply_patch" }

func (t *ApplyPatchTool) Description() string {
	return "Apply a Codex-style patch (*** Begin Patch ... *** End Patch) to files in the workspace. Prefer this over write_file for editing existing files."
}

func (t *ApplyPatchTool) Schema() map[string]any {
	return tool.ObjectSchema(map[string]any{
		"patch": map[string]any{
			"type":        "string",
			"description": "Full patch text in Codex apply_patch format",
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
		return "", fmt.Errorf("patch is required")
	}
	maxLines := t.Cfg.Tools.ApplyPatch.MaxChangedLines
	summary, err := patch.Apply(t.Perm.Workspace, in.Patch, t.Perm.ResolvePath, patch.ApplyOptions{
		MaxChangedLines: maxLines,
	})
	if err != nil {
		return "", err
	}
	return "applied: " + summary, nil
}

var _ tool.Tool = (*ApplyPatchTool)(nil)
