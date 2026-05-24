package globmatch_test

import (
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/tool/globmatch"
)

func TestHasMeta(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"internal/tool", false},
		{"internal/tool/*.go", true},
		{"**/*.go", true},
		{"file?.txt", true},
		{"[ab].go", true},
	}
	for _, tt := range tests {
		if got := globmatch.HasMeta(tt.path); got != tt.want {
			t.Errorf("HasMeta(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestSplitPath(t *testing.T) {
	tests := []struct {
		path         string
		wantBase     string
		wantPattern  string
	}{
		{"internal/tool", "internal/tool", ""},
		{"internal/tool/grep.go", "internal/tool/grep.go", ""},
		{"internal/tool/*.go", "internal/tool", "*.go"},
		{"internal/**/*.go", "internal", "**/*.go"},
		{"**/*.go", ".", "**/*.go"},
		{"*.go", ".", "*.go"},
	}
	for _, tt := range tests {
		base, pat := globmatch.SplitPath(tt.path)
		if base != tt.wantBase || pat != tt.wantPattern {
			t.Errorf("SplitPath(%q) = (%q, %q), want (%q, %q)", tt.path, base, pat, tt.wantBase, tt.wantPattern)
		}
	}
}
