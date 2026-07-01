package permissionmode

import (
	"fmt"
	"slices"
)

// Mode is the permission policy (readonly / ask / auto).
type Mode string

const (
	Readonly Mode = "readonly"
	Ask      Mode = "ask"
	Auto     Mode = "auto"
)

var configuredModes = []Mode{Readonly, Ask, Auto}

// ConfiguredStrings returns wire-format labels for explicit permission modes.
func ConfiguredStrings() []string {
	out := make([]string, len(configuredModes))
	for i, m := range configuredModes {
		out[i] = m.String()
	}
	return out
}

// String returns the wire-format permission mode label.
func (m Mode) String() string {
	return string(m)
}

// Valid reports whether m is a known permission mode (empty is allowed for unset).
func (m Mode) Valid() bool {
	return m == "" || m.Configured()
}

// Configured reports whether m is an explicit permission mode from configuredModes.
func (m Mode) Configured() bool {
	return slices.Contains(configuredModes, m)
}

// Parse parses a config, CLI, or DB string into Mode.
func Parse(s string) (Mode, error) {
	m := Mode(s)
	if m.Valid() {
		return m, nil
	}
	return "", fmt.Errorf("permissionmode: invalid mode %q", s)
}
