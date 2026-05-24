package mcp_test

import (
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/mcp"
	"github.com/wzhejunqiu/ds-code/internal/permission"
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
	if mcp.ClassifyPermission(tool) != permission.LevelHigh {
		t.Fatal("readOnlyHint must not downgrade write_file to low")
	}
}

func TestClassifyPermission_writeDestructive(t *testing.T) {
	destructive := true
	tool := mcpsdk.Tool{
		Name: "write_file",
		Annotations: mcpsdk.ToolAnnotation{
			DestructiveHint: &destructive,
		},
	}
	if mcp.ClassifyPermission(tool) != permission.LevelHighest {
		t.Fatalf("got %v, want LevelHighest", mcp.ClassifyPermission(tool))
	}
}

func TestClassifyPermission_writeNonDestructive(t *testing.T) {
	tool := mcpsdk.Tool{Name: "write_file"}
	if mcp.ClassifyPermission(tool) != permission.LevelHigh {
		t.Fatalf("got %v, want LevelHigh", mcp.ClassifyPermission(tool))
	}
}

func TestClassifyPermission_openWorld(t *testing.T) {
	open := true
	tool := mcpsdk.Tool{
		Name: "search",
		Annotations: mcpsdk.ToolAnnotation{
			OpenWorldHint: &open,
		},
	}
	if mcp.ClassifyPermission(tool) != permission.LevelMedium {
		t.Fatalf("got %v, want LevelMedium", mcp.ClassifyPermission(tool))
	}
}

func TestClassifyPermission_defaultMedium(t *testing.T) {
	tool := mcpsdk.Tool{Name: "read_file"}
	if mcp.ClassifyPermission(tool) != permission.LevelMedium {
		t.Fatalf("got %v, want LevelMedium", mcp.ClassifyPermission(tool))
	}
}

func TestIsWriteMCPToolName(t *testing.T) {
	if !mcp.IsWriteMCPToolName("create_item") {
		t.Fatal("create_item should be a write MCP tool")
	}
	if mcp.IsWriteMCPToolName("read_item") {
		t.Fatal("read_item should not be a write MCP tool")
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
