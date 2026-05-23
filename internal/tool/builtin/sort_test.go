package builtin_test

import (
	"testing"
	"time"

	"github.com/hejunqiu/ds-code/internal/tool/builtin"
)

func TestSortByModTimeDesc(t *testing.T) {
	oldTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	newTime := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	sameTime := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)

	type item struct {
		name string
		mod  time.Time
	}
	items := []item{
		{"old.txt", oldTime},
		{"new.txt", newTime},
		{"aaa.txt", sameTime},
		{"zzz.txt", sameTime},
	}
	builtin.SortByModTimeDesc(items,
		func(i item) time.Time { return i.mod },
		func(i item) string { return i.name },
	)
	if items[0].name != "new.txt" {
		t.Fatalf("newest first: got %q", items[0].name)
	}
	if items[1].name != "aaa.txt" || items[2].name != "zzz.txt" {
		t.Fatalf("tie-break by path asc: got %q, %q", items[1].name, items[2].name)
	}
	if items[3].name != "old.txt" {
		t.Fatalf("oldest last: got %q", items[3].name)
	}
}
