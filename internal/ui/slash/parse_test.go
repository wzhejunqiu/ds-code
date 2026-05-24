package slash_test

import (
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/ui/slash"
)

func TestParse(t *testing.T) {
	tests := []struct {
		line     string
		wantOK   bool
		wantCmd  string
		wantArgs string
	}{
		{"/help", true, "help", ""},
		{"/mode deepseek-v4-flash", true, "mode", "deepseek-v4-flash"},
		{"/btw question here", true, "btw", "question here"},
		{"  /clear", true, "clear", ""},
		{"请执行 /compact", false, "", ""},
		{"hello", false, "", ""},
		{"/", false, "", ""},
		{"/Git", false, "", ""},
	}
	for _, tt := range tests {
		cmd, args, ok := slash.Parse(tt.line)
		if ok != tt.wantOK || cmd != tt.wantCmd || args != tt.wantArgs {
			t.Errorf("Parse(%q) = (%q, %q, %v), want (%q, %q, %v)",
				tt.line, cmd, args, ok, tt.wantCmd, tt.wantArgs, tt.wantOK)
		}
	}
}
