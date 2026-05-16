package mcp

import (
	"encoding/json"

	mcpsdk "github.com/mark3labs/mcp-go/mcp"
)

func inputSchema(t mcpsdk.Tool, strict bool) map[string]any {
	if len(t.RawInputSchema) > 0 {
		var s map[string]any
		if err := json.Unmarshal(t.RawInputSchema, &s); err == nil && s != nil {
			if strict {
				s["additionalProperties"] = false
			}
			return s
		}
	}
	raw, err := json.Marshal(t.InputSchema)
	if err != nil {
		return map[string]any{"type": "object", "properties": map[string]any{}}
	}
	var s map[string]any
	if err := json.Unmarshal(raw, &s); err != nil || s == nil {
		return map[string]any{"type": "object", "properties": map[string]any{}}
	}
	if strict {
		s["additionalProperties"] = false
	}
	return s
}
