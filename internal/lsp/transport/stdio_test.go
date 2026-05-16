package transport

import (
	"bufio"
	"bytes"
	"encoding/json"
	"testing"
)

func TestReadWriteMessage_roundtrip(t *testing.T) {
	var buf bytes.Buffer
	payload := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
	}
	if err := WriteMessage(&buf, payload); err != nil {
		t.Fatal(err)
	}
	msg, err := ReadMessage(bufio.NewReader(&buf))
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(msg, &got); err != nil {
		t.Fatal(err)
	}
	if got["method"] != "initialize" {
		t.Fatalf("method = %v", got["method"])
	}
}
