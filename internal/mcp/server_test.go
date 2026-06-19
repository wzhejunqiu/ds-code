package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	mcpsdk "github.com/mark3labs/mcp-go/mcp"
	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/logging"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestFormatToolResult_text(t *testing.T) {
	res := &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{
			mcpsdk.TextContent{Text: "hello"},
			mcpsdk.TextContent{Text: "world"},
		},
	}
	got := formatToolResult(res)
	if got != "hello\nworld" {
		t.Fatalf("got %q", got)
	}
}

func TestFormatToolResult_errorFlag(t *testing.T) {
	res := &mcpsdk.CallToolResult{
		IsError: true,
		Content: []mcpsdk.Content{
			mcpsdk.TextContent{Text: "boom"},
		},
	}
	got := formatToolResult(res)
	if got != "error: boom" {
		t.Fatalf("got %q", got)
	}
}

func TestFormatToolResult_nil(t *testing.T) {
	if got := formatToolResult(nil); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestConnectServer_missingCommand(t *testing.T) {
	_, err := ConnectServer(context.Background(), config.MCPServerConfig{Name: "fs"}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMCPArgsPreview_truncates(t *testing.T) {
	args := []byte(`{"query":"` + strings.Repeat("x", 300) + `"}`)
	preview := mcpArgsPreview(args)
	if preview == "" {
		t.Fatal("expected non-empty preview")
	}
	if len(preview) > 203 {
		t.Fatalf("preview len %d > 200+ellipsis", len(preview))
	}
	if !strings.HasSuffix(preview, "...") {
		t.Fatalf("expected truncated preview: %q", preview)
	}
}

func TestLogMCPCall_argsPreview(t *testing.T) {
	core, logs := observer.New(zap.DebugLevel)
	restore := logging.ReplaceForTest(zap.New(core))
	defer restore()

	args := json.RawMessage(`{"query":"permission","limit":10}`)
	logMCPCall("graph", "semantic_search_nodes", args, len(args), 42, false, 0, nil)

	entries := logs.FilterMessage("mcp call tool")
	if entries.Len() != 1 {
		t.Fatalf("expected one log entry, got %d", entries.Len())
	}
	ctx := entries.All()[0].ContextMap()
	if ctx["args_preview"] == nil {
		t.Fatal("missing args_preview field")
	}
	if len(ctx["args_preview"].(string)) > 200 {
		t.Fatal("args_preview should be truncated to 200 chars")
	}
}
