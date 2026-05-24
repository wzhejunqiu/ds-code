package role_test

import (
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/role"
)

func TestRole_String(t *testing.T) {
	cases := []struct {
		r    role.Role
		want string
	}{
		{role.User, "user"},
		{role.Assistant, "assistant"},
		{role.Tool, "tool"},
		{role.System, "system"},
	}
	for _, tc := range cases {
		if got := tc.r.String(); got != tc.want {
			t.Fatalf("%q.String() = %q, want %q", tc.r, got, tc.want)
		}
	}
}
