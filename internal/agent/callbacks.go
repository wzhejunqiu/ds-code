package agent

// TurnCallbacks receives streaming and tool events during RunTurn.
// The TUI implements these in run.go (runTurnAsync → tea.Msg on deps.Events).
// All hooks are optional; omit a field to skip that UI update.
type TurnCallbacks struct {
	OnContentDelta        func(string) // assistant reply token(s)
	OnReasoningDelta      func(string) // thinking stream token(s)
	OnToolStart           func(name, args, command string)
	OnToolEnd             func(name, args, command, result string, isError bool)
	OnAssistantSegmentEnd func() // end of one assistant segment (before tools or next sub-round)
	// OnPlanningStart: round>0, after tools, before the next LLM HTTP request.
	OnPlanningStart func()
	// OnPlanningEnd: first content/reasoning delta, or LLM error/complete without stream.
	OnPlanningEnd func()
}
