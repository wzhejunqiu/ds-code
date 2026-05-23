// Package register wires built-in tools into a registry without import cycles.
package register

import (
	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/permission"
	"github.com/hejunqiu/ds-code/internal/tool"
	"github.com/hejunqiu/ds-code/internal/tool/builtin/glob"
	"github.com/hejunqiu/ds-code/internal/tool/builtin/grep"
	"github.com/hejunqiu/ds-code/internal/tool/builtin/list_dir"
	"github.com/hejunqiu/ds-code/internal/tool/builtin/read_file"
)

// ExploreTools adds read-only exploration tools for subagents and plan mode.
func ExploreTools(reg *tool.Registry, cfg *config.Config, perm *permission.Engine, gi *tool.GitignoreMatcher, strict bool) {
	reg.Register(&read_file.ReadFileTool{Cfg: cfg, Perm: perm, Strict: strict})
	reg.Register(&grep.GrepTool{Cfg: cfg, Perm: perm, Gitignore: gi, Strict: strict})
	reg.Register(&glob.GlobTool{Cfg: cfg, Perm: perm, Gitignore: gi, Strict: strict})
	reg.Register(&list_dir.ListDirTool{Cfg: cfg, Perm: perm, Gitignore: gi, Strict: strict})
}
