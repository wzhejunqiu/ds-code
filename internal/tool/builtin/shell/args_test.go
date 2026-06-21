package shell

import (
	"encoding/json"
	"testing"
)

func TestIsBackgroundArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args map[string]any
		want bool
	}{
		{
			name: "true when run_in_background set",
			args: map[string]any{"command": "echo hi", "run_in_background": true},
			want: true,
		},
		{
			name: "false when run_in_background omitted",
			args: map[string]any{"command": "echo hi"},
			want: false,
		},
		{
			name: "false when run_in_background explicitly false",
			args: map[string]any{"command": "echo hi", "run_in_background": false},
			want: false,
		},
		{
			name: "false for empty args",
			args: map[string]any{},
			want: false,
		},
		{
			name: "false for invalid json",
			args: nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var raw []byte
			if tt.args != nil {
				raw, _ = json.Marshal(tt.args)
			} else {
				raw = []byte("{not json")
			}
			if got := IsBackgroundArgs(raw); got != tt.want {
				t.Fatalf("IsBackgroundArgs(%q) = %v, want %v", raw, got, tt.want)
			}
		})
	}
}
