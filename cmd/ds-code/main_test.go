package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestRunRoot_nonTTYWithoutPrompt(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	oldIn := os.Stdin
	os.Stdin = r
	t.Cleanup(func() {
		os.Stdin = oldIn
		_ = r.Close()
	})
	_ = w.Close()

	cmd := newRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	err = runRoot(cmd, nil)
	if err != nil {
		t.Fatalf("runRoot: %v", err)
	}
	if got := out.String(); !strings.Contains(got, "not a TTY") {
		t.Fatalf("output = %q, want non-TTY hint", got)
	}
}

func TestNewRootCmd_hasSubcommands(t *testing.T) {
	cmd := newRootCmd()
	names := map[string]bool{}
	for _, c := range cmd.Commands() {
		names[c.Name()] = true
	}
	for _, want := range []string{"version", "sessions", "resume"} {
		if !names[want] {
			t.Fatalf("missing subcommand %q", want)
		}
	}
}
