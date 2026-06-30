package permission

import (
	"fmt"
	"testing"
)

func TestClassifyDeny(t *testing.T) {
	tests := []struct {
		err  error
		want string
	}{
		{ErrRejected, "user_rejected"},
		{ErrNeedTTY, "ask_no_tty"},
		{ErrDenied, "denied"},
		{fmt.Errorf("%w: web_fetch blocked", ErrDenied), "web_fetch"},
	}
	for _, tc := range tests {
		if got := classifyDeny(tc.err); got != tc.want {
			t.Fatalf("classifyDeny(%v) = %q, want %q", tc.err, got, tc.want)
		}
	}
}
