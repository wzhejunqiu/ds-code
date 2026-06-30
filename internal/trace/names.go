package trace

// SpanName is a typed OpenTelemetry span operation name.
type SpanName string

const (
	SpanRunTurn SpanName = "run_turn"
	SpanLLMChat SpanName = "llm.chat"
)

const (
	spanToolPrefix     = "tool."
	spanSubagentPrefix = "subagent."
)

// String returns the wire-format span name.
func (n SpanName) String() string {
	return string(n)
}

// SpanTool returns the span name for a built-in or MCP tool invocation.
func SpanTool(tool string) SpanName {
	return SpanName(spanToolPrefix + tool)
}

// SpanSubagent returns the span name for a sub-agent run.
func SpanSubagent(agentType string) SpanName {
	return SpanName(spanSubagentPrefix + agentType)
}
