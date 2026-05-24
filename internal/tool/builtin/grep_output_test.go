package builtin_test

import (
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/tool/builtin"
)

func TestParseGrepOutputMode(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"", builtin.GrepOutputFilesWithMatches, false},
		{"files_with_matches", builtin.GrepOutputFilesWithMatches, false},
		{"content", builtin.GrepOutputContent, false},
		{"count", builtin.GrepOutputCount, false},
		{"invalid", "", true},
	}
	for _, tc := range tests {
		got, err := builtin.ParseGrepOutputMode(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("ParseGrepOutputMode(%q) expected error", tc.in)
			}
			continue
		}
		if err != nil || got != tc.want {
			t.Fatalf("ParseGrepOutputMode(%q) = %q, %v; want %q", tc.in, got, err, tc.want)
		}
	}
}
