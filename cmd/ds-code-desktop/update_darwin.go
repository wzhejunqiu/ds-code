//go:build darwin

package main

import (
	"os/exec"
)

func checkForUpdates() {
	// Sparkle.framework integration is optional at build time; use CLI helper when present.
	if path := sparkleToolPath(); path != "" {
		_ = exec.Command(path, "check").Run()
		return
	}
	_ = exec.Command("open", defaultAppcastURL()).Run()
}

func sparkleToolPath() string {
	return ""
}

func defaultAppcastURL() string {
	return "https://example.com/ds-code/appcast.xml"
}
