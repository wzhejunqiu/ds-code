package slash_test

import (
	"testing"

	"github.com/hejunqiu/ds-code/internal/ui/slash"
)

func TestFilterCommands_prefix(t *testing.T) {
	cmds := slash.FilterCommands("/c")
	names := make(map[string]bool)
	for _, c := range cmds {
		names[c.Name] = true
	}
	for _, want := range []string{"clear", "compact", "checkpoint"} {
		if !names[want] {
			t.Fatalf("missing %q in %#v", want, names)
		}
	}
}
