// Command fakeserver is a minimal LSP stdio server for unit tests.
package main

import (
	"bufio"
	"encoding/json"
	"os"

	"github.com/wzhejunqiu/ds-code/internal/lsp/transport"
)

func main() {
	reader := bufio.NewReader(os.Stdin)
	writer := bufio.NewWriter(os.Stdout)

	for {
		msg, err := transport.ReadMessage(reader)
		if err != nil {
			return
		}
		var env struct {
			ID     *int            `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if err := json.Unmarshal(msg, &env); err != nil {
			continue
		}

		switch env.Method {
		case "initialize":
			if env.ID != nil {
				_ = transport.WriteMessage(writer, map[string]any{
					"jsonrpc": "2.0",
					"id":      *env.ID,
					"result": map[string]any{
						"capabilities": map[string]any{},
					},
				})
			}
		case "shutdown":
			if env.ID != nil {
				_ = transport.WriteMessage(writer, map[string]any{
					"jsonrpc": "2.0",
					"id":      *env.ID,
					"result":  map[string]any{},
				})
			}
		case "exit":
			return
		case "textDocument/didOpen":
			var params struct {
				TextDocument struct {
					URI string `json:"uri"`
				} `json:"textDocument"`
			}
			_ = json.Unmarshal(env.Params, &params)
			uri := params.TextDocument.URI
			if uri == "" {
				uri = "file:///test.go"
			}
			_ = transport.WriteMessage(writer, map[string]any{
				"jsonrpc": "2.0",
				"method":  "textDocument/publishDiagnostics",
				"params": map[string]any{
					"uri": uri,
					"diagnostics": []map[string]any{
						{
							"range": map[string]any{
								"start": map[string]any{"line": 0, "character": 0},
							},
							"severity": 1,
							"message":  "fake diagnostic",
						},
					},
				},
			})
		}
	}
}
