package permissionmode_test

import (
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/permissionmode"
)

func TestMode_Parse(t *testing.T) {
	for _, s := range []string{"readonly", "ask", "auto"} {
		m, err := permissionmode.Parse(s)
		if err != nil || m.String() != s {
			t.Fatalf("Parse(%q) = %q, %v", s, m, err)
		}
	}
	_, err := permissionmode.Parse("bogus")
	if err == nil {
		t.Fatal("expected error for bogus mode")
	}
}

func TestMode_Configured(t *testing.T) {
	if !permissionmode.Ask.Configured() {
		t.Fatal("ask should be configured")
	}
	if permissionmode.Mode("").Configured() {
		t.Fatal("empty should not be configured")
	}
}

func TestConfiguredStrings(t *testing.T) {
	got := permissionmode.ConfiguredStrings()
	want := []string{"readonly", "ask", "auto"}
	if len(got) != len(want) {
		t.Fatalf("ConfiguredStrings() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ConfiguredStrings()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
