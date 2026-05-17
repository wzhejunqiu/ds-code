//go:build !debug

package tui

// debugBeforeUpdate is a no-op in release builds (eligible for inlining).
func debugBeforeUpdate() {}
