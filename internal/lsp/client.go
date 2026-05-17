package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/lsp/transport"
)

// Diagnostic is a single LSP diagnostic item.
type Diagnostic struct {
	URI      string
	Line     int
	Col      int
	Severity string
	Message  string
}

// Client talks to one language server over stdio.
type Client struct {
	cfg    ServerConfig
	root   string
	lspCfg config.LSPConfig

	cmd    *exec.Cmd
	stdin  io.WriteCloser
	reader *bufio.Reader

	mu      sync.Mutex
	nextID  int
	diags   map[string][]Diagnostic
	pending map[int]chan json.RawMessage

	lastUsed time.Time
}

// NewClient creates a client (not started).
func NewClient(root string, lspCfg config.LSPConfig, srv ServerConfig) *Client {
	return &Client{
		cfg:     srv,
		root:    root,
		lspCfg:  lspCfg,
		diags:   make(map[string][]Diagnostic),
		pending: make(map[int]chan json.RawMessage),
	}
}

// Start launches the language server process.
func (c *Client) Start(ctx context.Context) error {
	c.mu.Lock()
	if c.cmd != nil {
		c.mu.Unlock()
		return nil
	}
	cmd := exec.CommandContext(ctx, c.cfg.Command, c.cfg.Args...)
	cmd.Dir = c.root
	cmd.Env = os.Environ()
	for k, v := range c.cfg.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		c.mu.Unlock()
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		c.mu.Unlock()
		return err
	}
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		c.mu.Unlock()
		return fmt.Errorf("lsp %s: start %s: %w", c.cfg.ID, c.cfg.Command, err)
	}
	c.cmd = cmd
	c.stdin = stdin
	c.reader = bufio.NewReader(stdout)
	go c.readLoop()
	c.mu.Unlock()

	if err := c.initialize(ctx); err != nil {
		c.mu.Lock()
		_ = c.closeLocked()
		c.mu.Unlock()
		return err
	}
	c.mu.Lock()
	c.lastUsed = time.Now()
	c.mu.Unlock()
	return nil
}

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
			"processId": os.Getpid(),
			"rootUri":   pathToURI(c.root),
			"capabilities": map[string]any{},
			"clientInfo": map[string]any{
				"name":    "ds-code",
				"version": "0.1",
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

// Close shuts down the language server.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closeLocked()
}

func (c *Client) closeLocked() error {
	if c.cmd == nil {
		return nil
	}
	if c.stdin != nil {
		id := c.allocIDLocked()
		ch := c.registerPendingLocked(id)
		_ = transport.WriteMessage(c.stdin, map[string]any{
			"jsonrpc": "2.0",
			"id":      id,
			"method":  "shutdown",
		})
		select {
		case <-ch:
		case <-time.After(2 * time.Second):
		}
		_ = transport.WriteMessage(c.stdin, map[string]any{
			"jsonrpc": "2.0",
			"method":  "exit",
		})
	}
	if c.cmd.Process != nil {
		_ = c.cmd.Process.Kill()
	}
	_ = c.cmd.Wait()
	c.cmd = nil
	return nil
}

func pathToURI(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	u := url.URL{Scheme: "file", Path: filepath.ToSlash(abs)}
	return u.String()
}

func languageID(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go":
		return "go"
	case ".ts", ".tsx":
		return "typescript"
	case ".js", ".jsx":
		return "javascript"
	case ".py":
		return "python"
	case ".java":
		return "java"
	case ".rs":
		return "rust"
	default:
		return "plaintext"
	}
}
