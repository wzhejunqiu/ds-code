package tui

// overlayKind is the active full-screen overlay above the chat transcript.
type overlayKind int

const (
	overlayNone overlayKind = iota // no overlay
	overlayContext                 // /context panel
	overlayHelp                    // ? shortcuts
	overlayComplete                // slash command completion
	overlayResume                  // /resume session picker
	overlayPrompt                  // tool permission ask
)
