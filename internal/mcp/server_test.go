package mcp

import (
	"context"
	"testing"

	"github.com/hejunqiu/ds-code/internal/config"
	mcpsdk "github.com/mark3labs/mcp-go/mcp"
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
	_, err := ConnectServer(context.Background(), config.MCPServerConfig{Name: "fs"})
	if err == nil {
		t.Fatal("expected error")
	}
}
