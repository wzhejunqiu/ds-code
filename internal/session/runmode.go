package session

import "github.com/wzhejunqiu/ds-code/internal/runmode"

// RunMode selects agent vs plan tool availability for a session.
type RunMode = runmode.RunMode

const (
	RunModeAgent = runmode.Agent
	RunModePlan  = runmode.Plan
)

// ParseRunMode parses a config or DB string into RunMode.
func ParseRunMode(s string) (RunMode, error) {
	return runmode.Parse(s)
}
