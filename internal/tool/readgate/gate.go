// Package readgate enforces read_file-before-apply_patch per session and sub-round.
package readgate

import (
	"context"
	"fmt"
	"strings"

	wspkg "github.com/wzhejunqiu/ds-code/internal/workspace"
)

type gateKey struct{}

// Gate tracks read_file paths for apply_patch validation within a sub-round.
type Gate struct {
	Workspace      string
	Snapshot       map[string]struct{} // canonical abs at sub-round start
	SameBatchReads map[string]struct{} // read_file paths in this sub-round's tool_calls
	mark           func(canonical string)
}

// NewGate builds a gate for one sub-round tool_calls batch.
func NewGate(workspace string, snapshot, sameBatchReads map[string]struct{}, mark func(string)) *Gate {
	if snapshot == nil {
		snapshot = map[string]struct{}{}
	}
	if sameBatchReads == nil {
		sameBatchReads = map[string]struct{}{}
	}
	return &Gate{
		Workspace:      workspace,
		Snapshot:       snapshot,
		SameBatchReads: sameBatchReads,
		mark:           mark,
	}
}

// WithGate attaches a read gate to ctx for the current sub-round.
func WithGate(ctx context.Context, g *Gate) context.Context {
	if g == nil {
		return ctx
	}
	return context.WithValue(ctx, gateKey{}, g)
}

// FromContext returns the read gate for the current tool execution.
func FromContext(ctx context.Context) (*Gate, bool) {
	if ctx == nil {
		return nil, false
	}
	g, ok := ctx.Value(gateKey{}).(*Gate)
	return g, ok && g != nil
}

// CanonicalPath resolves path under workspace to a stable absolute key.
func CanonicalPath(workspace, path string) (string, error) {
	if workspace == "" {
		return "", fmt.Errorf("readgate: empty workspace")
	}
	return wspkg.ResolveRel(workspace, path)
}

// MarkPath records a successful read_file for later sub-rounds.
func (g *Gate) MarkPath(path string) error {
	if g == nil || g.mark == nil {
		return nil
	}
	canon, err := CanonicalPath(g.Workspace, path)
	if err != nil {
		return err
	}
	g.mark(canon)
	return nil
}

// CheckApplyPatch validates required rel paths before apply_patch.
// sameBatchFmt and mustReadFmt are fmt.Sprintf templates with one %s for path list.
func (g *Gate) CheckApplyPatch(requiredRel []string, sameBatchFmt, mustReadFmt string) error {
	if g == nil {
		return nil
	}
	var sameBatch, missing []string
	seenSame := make(map[string]struct{})
	seenMissing := make(map[string]struct{})
	for _, rel := range requiredRel {
		canon, err := CanonicalPath(g.Workspace, rel)
		if err != nil {
			return err
		}
		if _, ok := g.SameBatchReads[canon]; ok {
			if _, dup := seenSame[rel]; !dup {
				seenSame[rel] = struct{}{}
				sameBatch = append(sameBatch, rel)
			}
			continue
		}
		if _, ok := g.Snapshot[canon]; !ok {
			if _, dup := seenMissing[rel]; !dup {
				seenMissing[rel] = struct{}{}
				missing = append(missing, rel)
			}
		}
	}
	if len(sameBatch) > 0 {
		return fmt.Errorf(sameBatchFmt, strings.Join(sameBatch, ", "))
	}
	if len(missing) > 0 {
		return fmt.Errorf(mustReadFmt, strings.Join(missing, ", "))
	}
	return nil
}
