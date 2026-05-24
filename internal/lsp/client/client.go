package client

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

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/lsp/transport"
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

// LastUsedAt returns when the client last handled a request.
func (c *Client) LastUsedAt() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.lastUsed
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
