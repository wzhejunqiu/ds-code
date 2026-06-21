package chattool

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/tool"
)

func TestRenderRunningBash_showsCountdown(t *testing.T) {
	now := time.Date(2026, 6, 21, 12, 0, 0, 0, time.UTC)
	deadline := now.Add(83 * time.Second)
	out := strings.Join(Render(Block{
		Name:            "bash",
		Args:            "Run tests",
		Command:         "go\x00go, test",
		Running:         true,
		TimeoutDeadline: deadline,
	}, 80, false, tool.DisplayContext{}, now), "\n")
	re := regexp.MustCompile(`\d:\d{2}|\d+s`)
	if !re.MatchString(out) {
		t.Fatalf("expected countdown in output:\n%s", out)
	}
}
