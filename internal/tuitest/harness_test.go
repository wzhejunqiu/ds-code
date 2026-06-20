//go:build tuitest

package tuitest

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/wzhejunqiu/ds-code/cmd/ds-code/slashcmd"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/input"
	tuimsg "github.com/wzhejunqiu/ds-code/internal/ui/tui/model/msg"
)

func TestHarness_toolPatchSingle(t *testing.T) {
	stack, m := runScenario(t, "tool-patch-single")
	if m.State.ErrLine != "" {
		t.Fatalf("ErrLine = %q", m.State.ErrLine)
	}
	assertFileContains(t, stack.Project, "sample.go", "// harness")
}

func TestHarness_toolPatchMulti(t *testing.T) {
	stack, m := runScenario(t, "tool-patch-multi")
	if m.State.ErrLine != "" {
		t.Fatalf("ErrLine = %q", m.State.ErrLine)
	}
	assertFileContains(t, stack.Project, "sample_multiline.go", "// harness")
}

func TestHarness_streamBasic(t *testing.T) {
	_, m := runScenario(t, "stream-basic")
	if m.State.ErrLine != "" {
		t.Fatalf("ErrLine = %q", m.State.ErrLine)
	}
	if !strings.Contains(chatText(m), "hello world") {
		t.Fatalf("chat = %q", chatText(m))
	}
}

func TestHarness_streamReasoning(t *testing.T) {
	_, m := runScenario(t, "stream-reasoning")
	if !strings.Contains(chatText(m), "answer") {
		t.Fatalf("chat = %q", chatText(m))
	}
}

func TestHarness_toolRead(t *testing.T) {
	_, m := runScenario(t, "tool-read")
	if m.State.ErrLine != "" {
		t.Fatalf("ErrLine = %q", m.State.ErrLine)
	}
	if !strings.Contains(chatText(m), "File read complete") {
		t.Fatalf("chat = %q", chatText(m))
	}
}

func TestHarness_errorAPI(t *testing.T) {
	_, m := runScenario(t, "error-api")
	if m.State.ErrLine == "" {
		t.Fatal("expected ErrLine for API error scenario")
	}
}

func TestHarness_errorContext(t *testing.T) {
	_, m := runScenario(t, "error-context")
	if m.State.ErrLine != "" {
		t.Fatalf("unexpected ErrLine: %q", m.State.ErrLine)
	}
	if !strings.Contains(chatText(m), "after compact") {
		t.Fatalf("chat = %q", chatText(m))
	}
}

func runScenario(t *testing.T, name string) (*Stack, *model.Model) {
	t.Helper()
	stack, err := NewStack(t)
	if err != nil {
		t.Fatal(err)
	}
	if err := stack.Registry.SetActive(name); err != nil {
		t.Fatal(err)
	}
	sc, ok := stack.Registry.Get(name)
	if !ok {
		t.Fatalf("scenario %q", name)
	}

	ctx := context.Background()
	sess, err := slashcmd.CreateSession(stack.Cfg, stack.Store)
	if err != nil {
		t.Fatal(err)
	}
	if err := slashcmd.SeedGitSnapshot(stack.Cfg, ctx, stack.Store, sess.ID); err != nil {
		t.Fatal(err)
	}

	events := make(chan tea.Msg, 256)
	deps := stack.Deps(sess.ID, func(context.Context, io.Writer, *string, string, string) (bool, error) {
		return false, nil
	})
	deps.Events = events

	m := model.New(&deps)
	m.State.Width = 120
	m.State.Height = 40

	done := make(chan struct{})
	go func() {
		defer close(done)
		for msg := range events {
			if msg == nil {
				continue
			}
			_, _ = m.Update(msg)
			if _, ok := msg.(tuimsg.TurnDoneMsg); ok {
				return
			}
		}
	}()

	noop := func() {}
	if cmd := input.SubmitLine(&m.State, sc.Prompt, noop, noop); cmd != nil {
		_ = cmd()
	}

	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("timeout waiting for turn")
	}
	return stack, m
}

func assertFileContains(t *testing.T, project, rel, want string) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(project, rel))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), want) {
		t.Fatalf("%s = %q, want substring %q", rel, b, want)
	}
}

func chatText(m *model.Model) string {
	var b strings.Builder
	if m.State.MainChat == nil {
		return ""
	}
	for _, blk := range m.State.MainChat {
		b.WriteString(blk.Content)
	}
	return b.String()
}
