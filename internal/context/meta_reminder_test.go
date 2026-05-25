package context

import (
	"strings"
	"testing"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/role"
	"github.com/wzhejunqiu/ds-code/internal/session"
)

func TestBuildMetaReminder(t *testing.T) {
	cfg := &config.Config{ProjectRoot: "/proj"}
	sess := session.Session{Model: "deepseek-v4-pro", RunMode: "agent"}
	now := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	got := BuildMetaReminder(sess, cfg, now)
	if !strings.Contains(got, "date: 2026-05-25") {
		t.Fatalf("missing date: %q", got)
	}
	if !strings.Contains(got, "project: /proj") {
		t.Fatalf("missing project: %q", got)
	}
}

func TestPrependMetaReminder(t *testing.T) {
	msgs := []llm.Message{{Role: role.User, Content: "hi"}}
	out := prependMetaReminder(msgs, "<system-reminder>meta</system-reminder>")
	if len(out) != 2 || !strings.Contains(out[0].Content, "system-reminder") {
		t.Fatalf("unexpected prepend: %+v", out)
	}
}

func TestAppendVerificationReminder(t *testing.T) {
	msgs := []llm.Message{{Role: role.User, Content: "check"}}
	out := appendVerificationReminder(msgs)
	if len(out) != 2 || !strings.Contains(out[1].Content, "VERDICT") {
		t.Fatalf("unexpected append: %+v", out)
	}
}
