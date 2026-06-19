//go:build tuitest

package tuitest

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wzhejunqiu/ds-code/cmd/ds-code/slashcmd"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/header"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model"
	tuimsg "github.com/wzhejunqiu/ds-code/internal/ui/tui/model/msg"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/view"
)

func TestHarness_headerNoticeAutoScroll(t *testing.T) {
	stack, err := NewStack(t)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	sess, err := slashcmd.CreateSession(stack.Cfg, stack.Store)
	if err != nil {
		t.Fatal(err)
	}
	if err := slashcmd.SeedGitSnapshot(stack.Cfg, ctx, stack.Store, sess.ID); err != nil {
		t.Fatal(err)
	}

	notices := make([]header.Notice, 0, 6)
	for i := 0; i < 6; i++ {
		notices = append(notices, header.Notice{
			Level: header.NoticeWarn,
			Text:  "NOTICE_" + string(rune('A'+i)) + " " + strings.Repeat("甲", 50),
		})
	}
	deps := stack.Deps(sess.ID, nil)
	deps.StartupNotices = notices

	const termW = 200
	m := model.New(&deps)
	next, initCmd := m.Update(tea.WindowSizeMsg{Width: termW, Height: 40})
	m = next.(*model.Model)

	zoneW := header.ZoneWidth(termW, false)
	if header.MaxScrollOffset(notices, zoneW) <= 0 {
		t.Fatal("expected scrollable startup notices")
	}
	if initCmd == nil {
		t.Fatal("expected resize to schedule notice scroll tick")
	}

	before := headerText(m)
	if !strings.Contains(before, "NOTICE_A") {
		t.Fatalf("initial header missing first notice: %q", before)
	}
	if m.State.NoticeScrollOffset != 0 {
		t.Fatalf("initial offset = %d, want 0", m.State.NoticeScrollOffset)
	}

	next, scrollCmd := m.Update(tuimsg.NoticeScrollTickMsg{})
	m = next.(*model.Model)
	if scrollCmd == nil {
		t.Fatal("expected follow-up notice scroll tick after advance")
	}
	if m.State.NoticeScrollOffset == 0 {
		t.Fatal("expected scroll offset to advance after tick")
	}

	after := headerText(m)
	if before == after {
		t.Fatalf("header unchanged after auto scroll:\n%s", after)
	}
}

func headerText(m *model.Model) string {
	innerW := m.State.Width - 2
	if innerW < 10 {
		innerW = 10
	}
	return view.ChatViewportContent(&m.State, innerW)
}
