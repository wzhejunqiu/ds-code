package agent

// TurnCallbacks receives streaming and tool events during RunTurn.
type TurnCallbacks struct {
	OnContentDelta        func(string)
	OnReasoningDelta      func(string)
	OnToolStart           func(name, args, command string)
	OnToolEnd             func(name, args, command, result string, isError bool)
	OnAssistantSegmentEnd func()
	// OnPlanningStart fires before the next LLM request after tools in a prior sub-round.
	OnPlanningStart func()
	// OnPlanningEnd fires when the next LLM response begins streaming or completes.
	OnPlanningEnd func()
}
