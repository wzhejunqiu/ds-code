package tui

import (
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/logging"
	"github.com/wzhejunqiu/ds-code/internal/mcp"
	"github.com/wzhejunqiu/ds-code/internal/session"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/header"
)

func TestHeaderWidths(t *testing.T) {
	cfg := &config.Config{ProjectRoot: "/tmp", LLM: config.LLMConfig{Model: "m"}}
	sess := &session.Session{ID: "x", Model: "m"}
	notices := []header.Notice{
		{Level: header.NoticeWarn, Text: logging.SensitiveDataWarningMsg},
		{Level: header.NoticeWarn, Text: header.FormatMCPSkippedSummary([]mcp.SkippedTool{
			{Server: "fs", Tool: "grep", Reason: mcp.SkipBuiltinConflict},
		})},
	}
	for w := 0; w <= 200; w++ {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("width %d panic: %v", w, r)
				}
			}()
			_ = header.Render(w, "v", cfg, sess, 0, nil, 0)
			_ = header.Render(w, "v", cfg, sess, 0, notices, 0)
			_ = header.Render(w, "v", cfg, sess, 0, notices, 2)
		}()
	}
}

func TestHeader_NoticesWideAndNarrow(t *testing.T) {
	cfg := &config.Config{ProjectRoot: "/tmp/proj", LLM: config.LLMConfig{Model: "m"}}
	sess := &session.Session{ID: "x", Model: "m"}
	notices := []header.Notice{{Level: header.NoticeWarn, Text: logging.SensitiveDataWarningMsg}}
	wide := header.Render(120, "0.1.1", cfg, sess, 0, notices, 0)
	if wide == "" {
		t.Fatal("expected wide header")
	}
	narrow := header.Render(40, "0.1.1", cfg, sess, 0, notices, 0)
	if narrow == "" {
		t.Fatal("expected narrow header")
	}
}
