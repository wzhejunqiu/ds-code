//go:build darwin

package sys

import (
	"fmt"
	"os/exec"
)

// Notify sends a macOS notification via osascript.
func Notify(title, body string) {
	script := fmt.Sprintf(`display notification %q with title %q`, body, title)
	_ = exec.Command("osascript", "-e", script).Run()
}

// SetDockBadge is updated via Hooks from the desktop entrypoint.
func SetDockBadge(count int) {}
