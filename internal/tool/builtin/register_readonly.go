package builtin

import (
	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/permission"
	"github.com/hejunqiu/ds-code/internal/tool"
)

// RegisterExploreTools adds read-only exploration tools for subagents and plan mode.
func RegisterExploreTools(reg *tool.Registry, cfg *config.Config, perm *permission.Engine, gi *tool.GitignoreMatcher, strict bool) {
	reg.Register(&ReadFileTool{Cfg: cfg, Perm: perm, Strict: strict})
	reg.Register(&GrepTool{Cfg: cfg, Perm: perm, Gitignore: gi, Strict: strict})
	reg.Register(&GlobTool{Cfg: cfg, Perm: perm, Gitignore: gi, Strict: strict})
	reg.Register(&ListDirTool{Cfg: cfg, Perm: perm, Gitignore: gi, Strict: strict})
}
