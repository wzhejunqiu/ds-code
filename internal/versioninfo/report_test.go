package versioninfo_test

import (
	"runtime"
	"strings"
	"testing"

	"github.com/hejunqiu/ds-code/internal/version"
	"github.com/hejunqiu/ds-code/internal/versioninfo"
)

func TestFormat_includesDSCodeVersion(t *testing.T) {
	out := versioninfo.Format(version.Version)
	if !strings.HasPrefix(out, "ds-code "+version.Version+"\n") {
		t.Fatalf("output:\n%s", out)
	}
}

func TestWrite_includesExplicitGitCommit(t *testing.T) {
	var b strings.Builder
	versioninfo.Write(&b, "1.0.0", "abc123def456")
	out := b.String()
	if !strings.Contains(out, "- git/commit: abc123def456\n") {
		t.Fatalf("output:\n%s", out)
	}
}

func TestFormat_includesPlatformLines(t *testing.T) {
	out := versioninfo.Format("1.2.3")
	required := []string{
		"- os/version:",
		"- os/kernel:",
		"- os/type: " + runtime.GOOS,
		"- os/arch:",
		"- go/version: " + runtime.Version(),
		"- go/linking:",
		"- go/tags:",
	}
	for _, want := range required {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}

func TestFormat_archDetail(t *testing.T) {
	out := versioninfo.Format("x")
	switch runtime.GOARCH {
	case "arm64":
		if !strings.Contains(out, "arm64 (ARMv8 compatible)") {
			t.Fatalf("output:\n%s", out)
		}
	case "amd64":
		if !strings.Contains(out, "amd64 (x86_64 compatible)") {
			t.Fatalf("output:\n%s", out)
		}
	}
}
