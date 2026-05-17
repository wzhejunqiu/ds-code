package versioninfo

import (
	"fmt"
	"io"
	"runtime"
	"runtime/debug"
	"strings"
)

// Write prints ds-code version and runtime/platform build metadata.
// gitCommit is the release-time commit id injected via -ldflags; when empty,
// git/commit is read from Go build metadata (vcs.revision).
func Write(w io.Writer, dsVersion, gitCommit string) {
	fmt.Fprintln(w, "ds-code "+dsVersion)
	for _, line := range lines(gitCommit) {
		fmt.Fprintf(w, "- %s\n", line)
	}
}

func lines(gitCommit string) []string {
	out := []string{
		"os/version: " + osVersionLine(),
		"os/kernel: " + osKernelLine(),
		"os/type: " + runtime.GOOS,
		"os/arch: " + archDetail(),
		"go/version: " + runtime.Version(),
		"go/linking: " + goLinking(),
		"go/tags: " + goTags(),
	}
	if commit := resolveGitCommit(gitCommit); commit != "" {
		out = append([]string{"git/commit: " + commit}, out...)
	}
	return out
}

func resolveGitCommit(ldflagsCommit string) string {
	if ldflagsCommit != "" {
		return ldflagsCommit
	}
	return gitCommitFromBuild()
}

func gitCommitFromBuild() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	var revision string
	modified := false
	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			revision = s.Value
		case "vcs.modified":
			modified = s.Value == "true"
		}
	}
	if revision == "" {
		return ""
	}
	if modified {
		return revision + " (dirty)"
	}
	return revision
}

func osVersionLine() string {
	ver := platformOSVersion()
	bit := bitness()
	if ver == "" {
		if bit != "" {
			return runtime.GOOS + " (" + bit + ")"
		}
		return runtime.GOOS
	}
	if bit != "" {
		return fmt.Sprintf("%s %s (%s)", runtime.GOOS, ver, bit)
	}
	return fmt.Sprintf("%s %s", runtime.GOOS, ver)
}

func osKernelLine() string {
	kernel := platformKernelVersion()
	if kernel == "" {
		return runtime.GOARCH
	}
	return fmt.Sprintf("%s (%s)", kernel, runtime.GOARCH)
}

func bitness() string {
	switch runtime.GOARCH {
	case "amd64", "arm64", "ppc64", "ppc64le", "riscv64", "s390x", "loong64":
		return "64 bit"
	case "386", "arm", "ppc", "riscv", "mips", "mipsle":
		return "32 bit"
	default:
		return ""
	}
}

func archDetail() string {
	switch runtime.GOARCH {
	case "arm64":
		return "arm64 (ARMv8 compatible)"
	case "amd64":
		return "amd64 (x86_64 compatible)"
	case "386":
		return "386 (x86 compatible)"
	default:
		return runtime.GOARCH
	}
}

func goLinking() string {
	if bi, ok := debug.ReadBuildInfo(); ok {
		for _, s := range bi.Settings {
			if s.Key == "-buildmode" && s.Value != "" && s.Value != "exe" {
				return s.Value
			}
		}
		for _, s := range bi.Settings {
			if s.Key == "CGO_ENABLED" {
				if s.Value == "0" {
					return "static"
				}
				return "dynamic"
			}
		}
	}
	// go test / pure Go builds without embedded settings default to static.
	return "static"
}

func goTags() string {
	if bi, ok := debug.ReadBuildInfo(); ok {
		for _, s := range bi.Settings {
			if s.Key == "-tags" && s.Value != "" {
				return s.Value
			}
		}
	}
	return "none"
}

// Format returns the full version report as a single string (for tests).
func Format(dsVersion string) string {
	var b strings.Builder
	Write(&b, dsVersion, "")
	return strings.TrimSuffix(b.String(), "\n")
}
