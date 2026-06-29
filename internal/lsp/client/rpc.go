package client

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/lsp/transport"
	"github.com/wzhejunqiu/ds-code/internal/version"
)

func (c *Client) readLoop() {
	for {
		msg, err := transport.ReadMessage(c.reader)
		if err != nil {
			return
		}
		var envelope struct {
			ID     *int            `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
			Result json.RawMessage `json:"result"`
			Error  *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(msg, &envelope); err != nil {
			continue
		}
		if envelope.ID != nil {
			c.mu.Lock()
			ch := c.pending[*envelope.ID]
			delete(c.pending, *envelope.ID)
			c.mu.Unlock()
			if ch != nil {
				ch <- msg
			}
			continue
		}
		if envelope.Method != "textDocument/publishDiagnostics" {
			continue
		}
		var params struct {
			URI         string `json:"uri"`
			Diagnostics []struct {
				Range struct {
					Start struct {
						Line      int `json:"line"`
						Character int `json:"character"`
					} `json:"start"`
				} `json:"range"`
				Severity int    `json:"severity"`
				Message  string `json:"message"`
			} `json:"diagnostics"`
		}
		if err := json.Unmarshal(envelope.Params, &params); err != nil {
			continue
		}
		var items []Diagnostic
		for _, d := range params.Diagnostics {
			items = append(items, Diagnostic{
				URI:      params.URI,
				Line:     d.Range.Start.Line + 1,
				Col:      d.Range.Start.Character + 1,
				Severity: severityName(d.Severity),
				Message:  d.Message,
			})
		}
		c.mu.Lock()
		c.diags[params.URI] = items
		c.mu.Unlock()
	}
}

func severityName(sev int) string {
	switch sev {
	case 1:
		return "error"
	case 2:
		return "warning"
	case 3:
		return "info"
	default:
		return "hint"
	}
}

func (c *Client) initialize(ctx context.Context) error {
	id := c.allocID()
	ch := c.registerPending(id)
	defer c.unregisterPending(id)

	req := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "initialize",
		"params": map[string]any{
			"processId":    os.Getpid(),
			"rootUri":      pathToURI(c.root),
			"capabilities": map[string]any{},
			"clientInfo": map[string]any{
				"name":    version.Name,
				"version": version.Version,
			},
		},
	}
	if err := transport.WriteMessage(c.stdin, req); err != nil {
		return err
	}
	if err := c.waitResponse(ctx, ch, id); err != nil {
		return err
	}
	return transport.WriteMessage(c.stdin, map[string]any{
		"jsonrpc": "2.0",
		"method":  "initialized",
		"params":  map[string]any{},
	})
}

func (c *Client) waitResponse(ctx context.Context, ch <-chan json.RawMessage, id int) error {
	timeout := c.lspCfg.DiagnosticsTimeout
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case msg := <-ch:
		var resp struct {
			Error *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(msg, &resp); err != nil {
			return err
		}
		if resp.Error != nil {
			return fmt.Errorf("lsp %s: %s", c.cfg.ID, resp.Error.Message)
		}
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("lsp %s: request %d timeout", c.cfg.ID, id)
	}
}

func (c *Client) allocID() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.allocIDLocked()
}

func (c *Client) allocIDLocked() int {
	c.nextID++
	return c.nextID
}

func (c *Client) registerPending(id int) chan json.RawMessage {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.registerPendingLocked(id)
}

func (c *Client) registerPendingLocked(id int) chan json.RawMessage {
	ch := make(chan json.RawMessage, 1)
	c.pending[id] = ch
	return ch
}

func (c *Client) unregisterPending(id int) {
	c.mu.Lock()
	delete(c.pending, id)
	c.mu.Unlock()
}

// OpenFile sends didOpen and waits for diagnostics until timeout.
func (c *Client) OpenFile(ctx context.Context, relPath string, content []byte, severityFilter map[string]bool, maxIssues int) ([]Diagnostic, error) {
	abs := filepath.Join(c.root, relPath)
	uri := pathToURI(abs)

	c.mu.Lock()
	c.lastUsed = time.Now()
	delete(c.diags, uri)
	c.mu.Unlock()

	params := map[string]any{
		"jsonrpc": "2.0",
		"method":  "textDocument/didOpen",
		"params": map[string]any{
			"textDocument": map[string]any{
				"uri":        uri,
				"languageId": languageID(relPath),
				"version":    1,
				"text":       string(content),
			},
		},
	}
	if err := transport.WriteMessage(c.stdin, params); err != nil {
		return nil, err
	}

	timeout := c.lspCfg.DiagnosticsTimeout
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-deadline.C:
			c.mu.Lock()
			items := filterDiags(c.diags[uri], severityFilter, maxIssues)
			c.mu.Unlock()
			return items, nil
		case <-tick.C:
			c.mu.Lock()
			raw := c.diags[uri]
			c.mu.Unlock()
			if len(raw) > 0 {
				return filterDiags(raw, severityFilter, maxIssues), nil
			}
		}
	}
}

func filterDiags(in []Diagnostic, sev map[string]bool, max int) []Diagnostic {
	var out []Diagnostic
	for _, d := range in {
		if len(sev) > 0 && !sev[d.Severity] {
			continue
		}
		out = append(out, d)
		if max > 0 && len(out) >= max {
			break
		}
	}
	return out
}
