package setup

import (
	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/lsp"
	"github.com/wzhejunqiu/ds-code/internal/llm"
	mcpsvc "github.com/wzhejunqiu/ds-code/internal/mcp"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/session/subagentstore"
	"github.com/wzhejunqiu/ds-code/internal/shelljobs/manager"
	"github.com/wzhejunqiu/ds-code/internal/tool"
	"github.com/wzhejunqiu/ds-code/internal/tool/register"
	"github.com/wzhejunqiu/ds-code/internal/tool/builtin/agent"
	"github.com/wzhejunqiu/ds-code/internal/tool/builtin/apply_patch"
	"github.com/wzhejunqiu/ds-code/internal/tool/builtin/diagnostics"
	"github.com/wzhejunqiu/ds-code/internal/tool/builtin/shell"
	"github.com/wzhejunqiu/ds-code/internal/tool/builtin/web_fetch"
	"github.com/wzhejunqiu/ds-code/internal/tool/builtin/write_file"
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
	ShellJobs *manager.Manager
	Subagent  subagentstore.Store
}

// RegisterReadOnly registers plan-mode and subagent tools.
func RegisterReadOnly(reg *tool.Registry, d Deps) {
	register.ExploreTools(reg, d.Cfg, d.Perm, d.Gitignore, d.Strict)
	if d.LSP != nil && d.Cfg.LSP.Enabled {
		reg.Register(&diagnostics.DiagnosticsTool{Cfg: d.Cfg, Perm: d.Perm, Gitignore: d.Gitignore, LSP: d.LSP, Strict: d.Strict})
	}
	if d.Cfg.Web.FetchEnabled {
		reg.Register(&web_fetch.WebFetchTool{Cfg: d.Cfg, Strict: d.Strict})
	}
}

// RegisterWrite registers mutating tools (agent mode only).
func RegisterWrite(reg *tool.Registry, d Deps) {
	reg.Register(&shell.ShellTool{Cfg: d.Cfg, Perm: d.Perm, Jobs: d.ShellJobs, Strict: d.Strict})
	reg.Register(&apply_patch.ApplyPatchTool{Cfg: d.Cfg, Perm: d.Perm, Strict: d.Strict})
	reg.Register(&write_file.WriteFileTool{Cfg: d.Cfg, Perm: d.Perm, Strict: d.Strict})
}

// RegisterAgentExtras registers the agent tool and MCP for full agent mode.
func RegisterAgentExtras(reg *tool.Registry, d Deps) {
	if d.LLM != nil {
		reg.Register(agent.NewAgentTool(d.Cfg, d.Perm, d.LLM, d.Strict, d.Subagent, reg))
	}
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
