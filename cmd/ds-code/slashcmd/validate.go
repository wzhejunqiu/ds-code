package slashcmd

import (
	"fmt"
	"strings"
)

// requireSessionEnv checks fields needed by most slash handlers.
func requireSessionEnv(env *Env) error {
	if env == nil {
		return fmt.Errorf("slashcmd: nil env")
	}
	if env.Out == nil {
		return fmt.Errorf("slashcmd: nil output writer")
	}
	if env.Cfg == nil {
		return fmt.Errorf("slashcmd: nil config")
	}
	if env.Store == nil {
		return fmt.Errorf("slashcmd: nil session store")
	}
	if env.SessionID == nil || strings.TrimSpace(*env.SessionID) == "" {
		return fmt.Errorf("slashcmd: session not set")
	}
	return nil
}
