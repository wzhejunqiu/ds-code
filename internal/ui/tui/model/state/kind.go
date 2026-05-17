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
)
