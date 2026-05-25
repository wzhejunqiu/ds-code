package context

import (
	"fmt"
	"strings"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/role"
	"github.com/wzhejunqiu/ds-code/internal/session"
)

const verificationReminder = `<system-reminder>VERIFICATION-ONLY: end with VERDICT: PASS | FAIL | PARTIAL</system-reminder>`

// BuildMetaReminder returns ephemeral session metadata for the API view (not persisted).
func BuildMetaReminder(sess session.Session, cfg *config.Config, now time.Time) string {
	if cfg == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString("<system-reminder>\n")
	fmt.Fprintf(&b, "date: %s\n", now.UTC().Format("2006-01-02"))
	fmt.Fprintf(&b, "project: %s\n", cfg.ProjectRoot)
	if sess.Model != "" {
		fmt.Fprintf(&b, "model: %s\n", sess.Model)
	}
	if sess.RunMode != "" {
		fmt.Fprintf(&b, "run_mode: %s\n", sess.RunMode)
	}
	b.WriteString("</system-reminder>")
	return b.String()
}

func prependMetaReminder(msgs []llm.Message, reminder string) []llm.Message {
	reminder = strings.TrimSpace(reminder)
	if reminder == "" {
		return msgs
	}
	out := make([]llm.Message, 0, len(msgs)+1)
	out = append(out, llm.Message{Role: role.User, Content: reminder})
	out = append(out, msgs...)
	return out
}

func appendVerificationReminder(msgs []llm.Message) []llm.Message {
	out := make([]llm.Message, len(msgs)+1)
	copy(out, msgs)
	out[len(msgs)] = llm.Message{Role: role.User, Content: verificationReminder}
	return out
}
