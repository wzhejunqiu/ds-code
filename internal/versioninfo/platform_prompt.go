package versioninfo

import (
	"fmt"
	"os"
	"strings"
)

// PlatformForPrompt returns OS/platform lines for system prompt (no ds-code/go/git metadata).
// Each element is a labeled line body without a leading "- " prefix.
func PlatformForPrompt() []string {
	out := []string{
		"操作系统：" + osVersionLine(),
		"内核/架构：" + kernelArchForPrompt(),
	}
	if shell := strings.TrimSpace(os.Getenv("SHELL")); shell != "" {
		out = append(out, "Shell："+shell)
	}
	return out
}

func kernelArchForPrompt() string {
	kernel := platformKernelVersion()
	arch := archDetail()
	if kernel == "" {
		return arch
	}
	return fmt.Sprintf("%s (%s)", kernel, arch)
}
