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
	// OnPlanningStart: before each LLM HTTP request after the first (round>0).
	// Round 0 planning is owned by the TUI on user submit.
	OnPlanningStart func()
	// OnPlanningEnd: first content/reasoning delta, or LLM complete/error without stream.
	OnPlanningEnd func()
	// Subagent hooks (task tool): nested exploration UI, separate from main chat tools.
	OnSubagentStart     func(id, label, prompt, agentType string, background bool)
	OnSubagentEnd       func(id, summary string, err error)
	OnSubagentToolStart func(id, name, args, command string)
	OnSubagentToolEnd   func(id, name, args, command, result string, isError bool)
}
