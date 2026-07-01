package config

import (
	"strings"
	"testing"
)

func TestForbiddenKeyHint(t *testing.T) {
	tests := []struct {
		key  string
		want string
	}{
		{"llm.api_key", "DS_CODE_DEEPSEEK_API_KEY"},
		{"session", "~/.ds-code/projects/"},
		{"session.db_path", "~/.ds-code/projects/"},
		{"audit.log_path", "audit.enabled"},
		{"checkpoint", "~/.ds-code/projects/"},
		{"unknown.key", "remove this key"},
	}
	for _, tt := range tests {
		got := forbiddenKeyHint(tt.key)
		if !strings.Contains(got, tt.want) {
			t.Fatalf("forbiddenKeyHint(%q) = %q, want substring %q", tt.key, got, tt.want)
		}
	}
}
