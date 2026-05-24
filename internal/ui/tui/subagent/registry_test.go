package subagent_test

import (
	"errors"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/ui/tui/subagent"
)

func TestRegistry_lifecycle(t *testing.T) {
	var reg subagent.Registry
	rec := reg.Start("sa-1", "probe", "read main.go", "Explore", false)
	if rec.ID != "sa-1" || reg.Len() != 1 {
		t.Fatalf("start: %+v len=%d", rec, reg.Len())
	}
	reg.ToolStart("sa-1", "read_file", "Read main.go", "")
	reg.ToolEnd("sa-1", "read_file", "Read main.go", "", "ok", false)
	reg.End("sa-1", "summary text", nil)
	got := reg.Get("sa-1")
	if got.Status != subagent.StatusDone {
		t.Fatalf("status=%v", got.Status)
	}
	reg.End("sa-2", "", errors.New("fail"))
	if reg.Get("sa-2") != nil {
		t.Fatal("expected missing sa-2")
	}
}
