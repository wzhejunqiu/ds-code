package state

// OverlayKind is which panel is shown below the input.
type OverlayKind int

const (
	OverlayNone OverlayKind = iota
	OverlayContext
	OverlayHelp
	OverlayComplete
	OverlayResume
	OverlayPrompt
	OverlaySubagentList
	OverlayTCase // harness: /tcase scenario picker (tuitest build)
)

// SubagentNav is the subagent manager view stack (main → list → detail).
type SubagentNav int

const (
	SubagentNavMain SubagentNav = iota
	SubagentNavList
	SubagentNavDetail
)
