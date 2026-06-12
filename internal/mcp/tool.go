package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	mcpsdk "github.com/mark3labs/mcp-go/mcp"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/tool"
)

// adapterTool implements tool.Tool for one MCP tool.
type adapterTool struct {
	server     *Server
	mcpTool    string
	normalized string
	desc       string
	schema     map[string]any
	level      permission.Level
	deferLoad  bool
}

func newAdapterTool(srv *Server, t mcpsdk.Tool, strict bool, deferLoad bool) *adapterTool {
	return &adapterTool{
		server:     srv,
		mcpTool:    t.Name,
		normalized: ToolName(srv.Name, t.Name),
		desc:       t.Description,
		schema:     inputSchema(t, strict),
		level:      ClassifyPermission(t),
		deferLoad:  deferLoad,
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

func (a *adapterTool) ShouldDefer() bool { return a.deferLoad }

func (a *adapterTool) StubSchema() map[string]any {
	return tool.ObjectSchema(map[string]any{
		"_note": map[string]any{
			"type":        "string",
			"description": "Deferred MCP tool. Call tool_search before use.",
		},
	}, nil, false)
}

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
