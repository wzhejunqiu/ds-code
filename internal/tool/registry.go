package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/logging"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"go.uber.org/zap"
)

// Tool is a built-in agent tool.
type Tool interface {
	Name() string
	Description() string
	Schema() map[string]any
	Execute(ctx context.Context, args json.RawMessage) (string, error)
	PermissionLevel() permission.Level
}

// Registry holds registered tools.
type Registry struct {
	tools map[string]Tool
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

// Register adds a tool.
func (r *Registry) Register(t Tool) {
	r.tools[t.Name()] = t
}

// Get returns a tool by name.
func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

// Definitions returns tool defs for the LLM API (deferred tools use stub schemas).
func (r *Registry) Definitions() []llm.ToolDef {
	names := make([]string, 0, len(r.tools))
	for n := range r.tools {
		names = append(names, n)
	}
	sort.Strings(names)
	out := make([]llm.ToolDef, 0, len(names))
	for _, n := range names {
		t := r.tools[n]
		out = append(out, llm.ToolDef{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters:  r.parametersForLLM(t),
		})
	}
	return out
}

// FullSchema returns the complete JSON schema for a tool (ignores defer stubs).
func (r *Registry) FullSchema(name string) (map[string]any, bool) {
	t, ok := r.Get(name)
	if !ok {
		return nil, false
	}
	return t.Schema(), true
}

func (r *Registry) parametersForLLM(t Tool) map[string]any {
	if dt, ok := t.(DeferredTool); ok && dt.ShouldDefer() {
		return dt.StubSchema()
	}
	return t.Schema()
}

// HasDeferredTools reports whether any registered tool uses deferred loading.
func (r *Registry) HasDeferredTools() bool {
	for _, t := range r.tools {
		if dt, ok := t.(DeferredTool); ok && dt.ShouldDefer() {
			return true
		}
	}
	return false
}

// Execute runs a tool by name.
func (r *Registry) Execute(ctx context.Context, name string, args json.RawMessage) (string, error) {
	t, ok := r.Get(name)
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	start := time.Now()
	out, err := t.Execute(ctx, args)
	logging.L().Debug("tool registry execute",
		zap.String("tool", name),
		zap.Int("args_chars", len(args)),
		zap.Int("result_chars", len(out)),
		zap.Int64("duration_ms", time.Since(start).Milliseconds()),
		zap.Bool("ok", err == nil),
	)
	if err != nil {
		return out, err
	}
	return out, nil
}

// All returns all registered tools (unordered).
func (r *Registry) All() []Tool {
	out := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, t)
	}
	return out
}

// Subset returns a new registry containing only the named tools. Missing names are ignored.
func (r *Registry) Subset(names []string) *Registry {
	s := NewRegistry()
	nameSet := make(map[string]bool, len(names))
	for _, n := range names {
		nameSet[n] = true
	}
	for _, t := range r.tools {
		if nameSet[t.Name()] {
			s.Register(t)
		}
	}
	return s
}

// ArgsMap unmarshals tool arguments to a map for permission checks.
func ArgsMap(args json.RawMessage) map[string]any {
	var m map[string]any
	_ = json.Unmarshal(args, &m)
	if m == nil {
		m = make(map[string]any)
	}
	return m
}
