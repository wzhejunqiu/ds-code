package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/wzhejunqiu/ds-code/internal/permission"
	mcpsdk "github.com/mark3labs/mcp-go/mcp"
)

// adapterTool implements tool.Tool for one MCP tool.
type adapterTool struct {
	server     *Server
	mcpTool    string
	normalized string
	desc       string
	schema     map[string]any
	level      permission.Level
}

func newAdapterTool(srv *Server, t mcpsdk.Tool, strict bool) *adapterTool {
	return &adapterTool{
		server:     srv,
		mcpTool:    t.Name,
		normalized: ToolName(srv.Name, t.Name),
		desc:       t.Description,
		schema:     inputSchema(t, strict),
		level:      ClassifyPermission(t),
	}
}

func (a *adapterTool) Name() string { return a.normalized }

func (a *adapterTool) Description() string {
	if a.desc == "" {
		return fmt.Sprintf("MCP tool %s on server %s", a.mcpTool, a.server.Name)
	}
	return fmt.Sprintf("[MCP %s] %s", a.server.Name, a.desc)
}

func (a *adapterTool) Schema() map[string]any { return a.schema }

func (a *adapterTool) PermissionLevel() permission.Level { return a.level }

func (a *adapterTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var result string
	var callErr error

	func() {
		defer func() {
			if r := recover(); r != nil {
				callErr = fmt.Errorf("mcp: server %q panic: %v", a.server.Name, r)
			}
		}()
		result, callErr = a.server.CallTool(ctx, a.mcpTool, args)
	}()

	if callErr != nil {
		return "", callErr
	}
	return result, nil
}
