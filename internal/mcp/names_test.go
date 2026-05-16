package mcp_test

import (
	"testing"

	"github.com/hejunqiu/ds-code/internal/mcp"
)

func TestToolName_roundTrip(t *testing.T) {
	name := mcp.ToolName("fs", "read_file")
	srv, tool, ok := mcp.ParseToolName(name)
	if !ok || srv != "fs" || tool != "read_file" {
		t.Fatalf("parse %q => %q %q %v", name, srv, tool, ok)
	}
}

func TestValidateServerName(t *testing.T) {
	if err := mcp.ValidateServerName("my-server"); err != nil {
		t.Fatal(err)
	}
	if err := mcp.ValidateServerName("Bad"); err == nil {
		t.Fatal("expected error for uppercase")
	}
}
