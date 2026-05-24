package tool

// ReadOnlyTool is implemented by tools that never mutate state.
type ReadOnlyTool interface {
	IsReadOnly() bool
}

// ConcurrencySafeTool is implemented by tools safe to run concurrently with peers.
type ConcurrencySafeTool interface {
	IsConcurrencySafe() bool
}

// IsToolReadOnly reports whether t is a read-only tool.
func IsToolReadOnly(t Tool) bool {
	if r, ok := t.(ReadOnlyTool); ok {
		return r.IsReadOnly()
	}
	return false
}

// IsToolConcurrencySafe reports whether t can run concurrently.
func IsToolConcurrencySafe(t Tool) bool {
	if c, ok := t.(ConcurrencySafeTool); ok {
		return c.IsConcurrencySafe()
	}
	return false
}
