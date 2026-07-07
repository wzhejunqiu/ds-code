package sys_test

import (
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/desktop/sys"
)

func TestEnsurePATH_prependsHomebrew(t *testing.T) {
	t.Setenv("PATH", "/usr/bin")
	sys.EnsurePATH()
	p := sys.ExtendedPATH()
	if !strings.Contains(p, "/opt/homebrew/bin") {
		t.Fatalf("PATH missing homebrew: %s", p)
	}
}

func TestCheckDependencies_returnsKnownTools(t *testing.T) {
	deps := sys.CheckDependencies()
	if len(deps) < 3 {
		t.Fatalf("deps = %d", len(deps))
	}
	names := map[string]bool{}
	for _, d := range deps {
		names[d.Name] = true
	}
	for _, want := range []string{"git", "node", "gopls"} {
		if !names[want] {
			t.Fatalf("missing %s", want)
		}
	}
}

func TestBadgeCount(t *testing.T) {
	sys.TurnStarted()
	sys.PermissionWaiting(true)
	if sys.BadgeCount() < 2 {
		t.Fatalf("badge = %d", sys.BadgeCount())
	}
	sys.TurnFinished("", "", false)
	sys.PermissionWaiting(false)
}
