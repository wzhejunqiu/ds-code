package mcp

import (
	"encoding/json"
	"testing"

	mcpsdk "github.com/mark3labs/mcp-go/mcp"
)

func TestInputSchema_rawInputStrict(t *testing.T) {
	raw := json.RawMessage(`{"type":"object","properties":{"x":{"type":"string"}}}`)
	tool := mcpsdk.Tool{RawInputSchema: raw}
	s := inputSchema(tool, true)
	if s["additionalProperties"] != false {
		t.Fatalf("schema = %v", s)
	}
}

func TestInputSchema_fallbackObject(t *testing.T) {
	tool := mcpsdk.Tool{}
	s := inputSchema(tool, false)
	if _, ok := s["properties"]; !ok {
		t.Fatalf("schema = %v", s)
	}
}
