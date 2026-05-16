package tool

// ObjectSchema builds a JSON Schema object for tool parameters.
func ObjectSchema(properties map[string]any, required []string, strict bool) map[string]any {
	s := map[string]any{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		s["required"] = required
	}
	if strict {
		s["additionalProperties"] = false
	}
	return s
}
