package setup

import (
	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/lsp"
	"github.com/hejunqiu/ds-code/internal/llm"
	mcpsvc "github.com/hejunqiu/ds-code/internal/mcp"
	"github.com/hejunqiu/ds-code/internal/permission"
	"github.com/hejunqiu/ds-code/internal/session/subagentstore"
	"github.com/hejunqiu/ds-code/internal/shelljobs/manager"
	"github.com/hejunqiu/ds-code/internal/tool"
	"github.com/hejunqiu/ds-code/internal/tool/builtin"
)

// Deps holds shared dependencies for tool registration.
type Deps struct {
	Cfg       *config.Config
	Perm      *permission.Engine
	Gitignore *tool.GitignoreMatcher
	Strict    bool
	LLM       llm.Client
	LSP       *lsp.Manager
	MCP       *mcpsvc.Manager
	ShellJobs  *manager.Manager
	Subagent   subagentstore.Store
}

// RegisterReadOnly registers plan-mode and subagent tools.
func RegisterReadOnly(reg *tool.Registry, d Deps) {
	builtin.RegisterExploreTools(reg, d.Cfg, d.Perm, d.Gitignore, d.Strict)
	if d.LSP != nil && d.Cfg.LSP.Enabled {
		reg.Register(&builtin.DiagnosticsTool{Cfg: d.Cfg, Perm: d.Perm, Gitignore: d.Gitignore, LSP: d.LSP, Strict: d.Strict})
	}
	if d.Cfg.Web.FetchEnabled {
		reg.Register(&builtin.WebFetchTool{Cfg: d.Cfg, Strict: d.Strict})
	}
}

// RegisterWrite registers mutating tools (agent mode only).
func RegisterWrite(reg *tool.Registry, d Deps) {
	reg.Register(&builtin.ShellTool{Cfg: d.Cfg, Perm: d.Perm, Jobs: d.ShellJobs, Strict: d.Strict})
	reg.Register(&builtin.ApplyPatchTool{Cfg: d.Cfg, Perm: d.Perm, Strict: d.Strict})
	reg.Register(&builtin.WriteFileTool{Cfg: d.Cfg, Perm: d.Perm, Strict: d.Strict})
}

// RegisterAgentExtras registers task, web_search, and MCP for full agent mode.
func RegisterAgentExtras(reg *tool.Registry, d Deps) {
	if d.LLM != nil {
		reg.Register(builtin.NewTaskTool(d.Cfg, d.Perm, d.LLM, d.Strict, d.Subagent))
	}
	// web_search remains disabled until a search provider is wired (see human tool display plan).
	_ = d.Cfg.Web.SearchEnabled
	if d.MCP != nil {
		d.MCP.Register(reg)
	}
}

// BuildRegistry creates a tool registry for the given run mode.
func BuildRegistry(runMode string, d Deps) *tool.Registry {
	reg := tool.NewRegistry()
	RegisterReadOnly(reg, d)
	if runMode == "plan" {
		return reg
	}
	RegisterWrite(reg, d)
	RegisterAgentExtras(reg, d)
	return reg
}
