package ui

import "time"

// ExitConfirmTimeout is how long Ctrl+C / Ctrl+D double-press exit stays armed.
const ExitConfirmTimeout = 5 * time.Second
