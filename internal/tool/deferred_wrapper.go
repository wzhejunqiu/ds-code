package tool

import (
	"context"
	"encoding/json"

	"github.com/wzhejunqiu/ds-code/internal/permission"
)

// deferredWrapper exposes a minimal schema to the LLM; full schema via tool_search.
type deferredWrapper struct {
	inner Tool
}

// WrapDeferred returns a DeferredTool wrapper around t.
func WrapDeferred(t Tool) DeferredTool {
	return &deferredWrapper{inner: t}
}

func (d *deferredWrapper) Name() string        { return d.inner.Name() }
func (d *deferredWrapper) Description() string { return d.inner.Description() }
func (d *deferredWrapper) Schema() map[string]any {
	return d.inner.Schema()
}
func (d *deferredWrapper) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	return d.inner.Execute(ctx, args)
}
func (d *deferredWrapper) PermissionLevel() permission.Level {
	return d.inner.PermissionLevel()
}

func (d *deferredWrapper) ShouldDefer() bool { return true }

func (d *deferredWrapper) StubSchema() map[string]any {
	return ObjectSchema(map[string]any{
		"_note": map[string]any{
			"type":        "string",
			"description": "Full parameters omitted. Call tool_search with this tool name before invoking.",
		},
	}, nil, false)
}
