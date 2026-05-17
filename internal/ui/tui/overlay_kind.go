package tui

// overlayKind is which panel is shown below the input (see README.md).
type overlayKind int

const (
	overlayNone overlayKind = iota // no overlay
	overlayContext                 // /context panel
	overlayHelp                    // ? shortcuts
	overlayComplete                // slash command completion
	overlayResume                  // /resume session picker
	overlayPrompt                  // tool permission ask
)
