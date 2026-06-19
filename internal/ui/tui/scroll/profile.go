package scroll

import (
	"os"
	"strings"
)

// Profile classifies the terminal for scroll drain behavior.
type Profile int

const (
	ProfileNative Profile = iota
	ProfileIntegrated
)

// DetectProfile returns the scroll profile for the current terminal environment.
func DetectProfile() Profile {
	if os.Getenv("VSCODE_INJECTED") == "1" {
		return ProfileIntegrated
	}
	switch strings.ToLower(os.Getenv("TERM_PROGRAM")) {
	case "vscode", "cursor":
		return ProfileIntegrated
	default:
		return ProfileNative
	}
}
