package slashcmd

import (
	"fmt"
	"strings"

	"github.com/wzhejunqiu/ds-code/internal/agent/spawn"
)

func Remember(env *Env, args string) error {
	parts := strings.SplitN(strings.TrimSpace(args), " ", 2)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("usage: /remember <user|feedback|project|reference> <text>")
	}
	agentType := "general-purpose"
	if env.ActiveAgentType != "" {
		agentType = env.ActiveAgentType
	}
	if err := spawn.SaveAgentMemory(agentType, parts[0], parts[1]); err != nil {
		return err
	}
	fmt.Fprintf(env.Out, "Saved to agent memory (%s/%s.md).\n", agentType, strings.TrimSuffix(parts[0], ".md"))
	return nil
}
