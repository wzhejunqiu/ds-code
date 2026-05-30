package runmode

import "fmt"

// RunMode selects agent vs plan tool availability.
type RunMode string

const (
	Agent RunMode = "agent"
	Plan  RunMode = "plan"
)

// String returns the wire-format run mode label.
func (m RunMode) String() string {
	return string(m)
}

// Valid reports whether m is a known run mode (empty is allowed for unset session rows).
func (m RunMode) Valid() bool {
	return m == "" || m == Agent || m == Plan
}

// Configured reports whether m is an explicit agent or plan mode (required for config).
func (m RunMode) Configured() bool {
	return m == Agent || m == Plan
}

// Parse parses a config, CLI, or DB string into RunMode.
func Parse(s string) (RunMode, error) {
	m := RunMode(s)
	if m.Valid() {
		return m, nil
	}
	return "", fmt.Errorf("runmode: invalid run_mode %q", s)
}
