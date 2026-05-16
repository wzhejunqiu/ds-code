package mcp_test

import (
	"testing"

	"github.com/hejunqiu/ds-code/internal/mcp"
	"github.com/hejunqiu/ds-code/internal/permission"
	mcpsdk "github.com/mark3labs/mcp-go/mcp"
)

func TestClassifyPermission_readOnlyHint(t *testing.T) {
	ro := true
	tool := mcpsdk.Tool{
		Name: "write_file",
		Annotations: mcpsdk.ToolAnnotation{
			ReadOnlyHint: &ro,
		},
	}
	if mcp.ClassifyPermission(tool) != permission.LevelLow {
		t.Fatal("expected low for readOnlyHint")
	}
}

func TestIsWriteTool_normalized(t *testing.T) {
	name := mcp.ToolName("fs", "write_file")
	if !mcp.IsWriteTool(name) {
		t.Fatal("write_file should be write")
	}
	readName := mcp.ToolName("fs", "read_file")
	if mcp.IsWriteTool(readName) {
		t.Fatal("read_file should not be write")
	}
}
