package header

import "testing"

func TestAdvanceScrollOffset_wraps(t *testing.T) {
	var notices []Notice
	for i := 0; i < 12; i++ {
		notices = append(notices, Notice{Level: NoticeInfo, Text: "line " + string(rune('a'+i)) + " " + repeat("x", 30)})
	}
	termW := 120
	if MaxScrollOffset(notices, ZoneWidth(termW, false)) <= 0 {
		t.Fatal("expected scrollable notices")
	}
	var offset int
	maxOff := MaxScrollOffset(notices, ZoneWidth(termW, false))
	for i := 0; i < maxOff+1; i++ {
		if !AdvanceScrollOffset(notices, termW, false, &offset) {
			t.Fatalf("advance failed at step %d", i)
		}
	}
	if offset != 0 {
		t.Fatalf("expected wrap to 0, got %d", offset)
	}
}

func TestNeedsAutoScroll(t *testing.T) {
	short := []Notice{{Level: NoticeWarn, Text: "ok"}}
	if NeedsAutoScroll(short, 120, false) {
		t.Fatal("short notice should not auto-scroll")
	}
	var long []Notice
	for i := 0; i < 10; i++ {
		long = append(long, Notice{Level: NoticeWarn, Text: repeat("通知", 20)})
	}
	if !NeedsAutoScroll(long, 120, false) {
		t.Fatal("long notices should auto-scroll")
	}
}

func repeat(s string, n int) string {
	out := make([]byte, 0, len(s)*n)
	for i := 0; i < n; i++ {
		out = append(out, s...)
	}
	return string(out)
}
