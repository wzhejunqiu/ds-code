package agent

import (
	"encoding/json"
	"strings"

	"github.com/wzhejunqiu/ds-code/internal/tool"
)

// preToolUseOutput is optional JSON from a PreToolUse hook stdout.
type preToolUseOutput struct {
	Command string         `json:"command,omitempty"`
	Args    map[string]any `json:"args,omitempty"`
}

// applyPreToolUseResults merges hook stdout into tool arguments when valid JSON is returned.
func applyPreToolUseResults(toolName string, rawArgs []byte, results []HookResult) []byte {
	for _, res := range results {
		if res.Error != nil {
			continue
		}
		out := strings.TrimSpace(res.Output)
		if out == "" {
			continue
		}
		var patch preToolUseOutput
		if err := json.Unmarshal([]byte(out), &patch); err != nil {
			continue
		}
		var args map[string]any
		_ = json.Unmarshal(rawArgs, &args)
		if args == nil {
			args = make(map[string]any)
		}
		changed := false
		if patch.Command != "" && tool.NameShell.Matches(toolName) {
			args["command"] = patch.Command
			changed = true
		}
		if len(patch.Args) > 0 {
			for k, v := range patch.Args {
				args[k] = v
			}
			changed = true
		}
		if !changed {
			continue
		}
		b, err := json.Marshal(args)
		if err != nil {
			continue
		}
		return b
	}
	return rawArgs
}
