package tool

import (
	"github.com/wzhejunqiu/ds-code/internal/permission"
)

// PermBound is implemented by tools that carry a permission engine for workspace paths.
type PermBound interface {
	Tool
	WithPerm(perm *permission.Engine) Tool
}

// RebindRegistryPerm returns a new registry with PermBound tools rebound to perm.
// Deferred tools are unwrapped, rebound, and re-wrapped.
func RebindRegistryPerm(reg *Registry, perm *permission.Engine) *Registry {
	if reg == nil || perm == nil {
		return reg
	}
	out := NewRegistry()
	for _, t := range reg.All() {
		rebound := rebindTool(t, perm)
		if server, ok := reg.MCPServerForTool(t.Name()); ok {
			out.RegisterMCPTool(rebound, server)
		} else {
			out.Register(rebound)
		}
	}
	return out
}

func rebindTool(t Tool, perm *permission.Engine) Tool {
	if dt, ok := t.(DeferredTool); ok && dt.ShouldDefer() {
		if dw, ok := t.(*deferredWrapper); ok {
			return WrapDeferred(rebindTool(dw.inner, perm))
		}
	}
	if pb, ok := t.(PermBound); ok {
		return pb.WithPerm(perm)
	}
	return t
}
