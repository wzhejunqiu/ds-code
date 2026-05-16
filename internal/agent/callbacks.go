package agent

// TurnCallbacks receives streaming and tool events during RunTurn.
type TurnCallbacks struct {
	OnContentDelta   func(string)
	OnReasoningDelta func(string)
	OnToolStart      func(name string)
	OnToolEnd        func(name, preview string)
}
