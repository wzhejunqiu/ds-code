// Package register wires built-in tools into a registry without import cycles.
package register

import (
	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/tool"
	"github.com/wzhejunqiu/ds-code/internal/tool/builtin/glob"
	"github.com/wzhejunqiu/ds-code/internal/tool/builtin/grep"
	"github.com/wzhejunqiu/ds-code/internal/tool/builtin/read_file"
	"github.com/wzhejunqiu/ds-code/internal/tool/searchskip"
)

// ExploreTools adds read-only exploration tools for subagents and plan mode.
func ExploreTools(reg *tool.Registry, cfg *config.Config, perm *permission.Engine, searchSkip *searchskip.Matcher, strict bool) {
	reg.Register(&read_file.ReadFileTool{Cfg: cfg, Perm: perm, Strict: strict})
	reg.Register(&grep.GrepTool{Cfg: cfg, Perm: perm, SearchSkip: searchSkip, Strict: strict})
	reg.Register(&glob.GlobTool{Cfg: cfg, Perm: perm, SearchSkip: searchSkip, Strict: strict})
}
