package tui

import (
	"testing"

	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/session"
)

func TestHeaderWidths(t *testing.T) {
	cfg := &config.Config{ProjectRoot: "/tmp", LLM: config.LLMConfig{Model: "m"}}
	sess := &session.Session{ID: "x", Model: "m"}
	for w := 0; w <= 200; w++ {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("width %d panic: %v", w, r)
				}
			}()
			_ = renderHeader(w, "v", cfg, sess)
		}()
	}
}
