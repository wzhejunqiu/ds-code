package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/security"
	mcpclient "github.com/mark3labs/mcp-go/client"
	mcpsdk "github.com/mark3labs/mcp-go/mcp"
)

// Server is a connected MCP server subprocess.
type Server struct {
	Name   string
	client *mcpclient.Client

	// Test hooks (nil = use client). Set only in unit tests.
	testListTools func(context.Context) ([]mcpsdk.Tool, error)
	testCallTool  func(context.Context, string, json.RawMessage) (string, error)
	testClose     func() error
}

// ConnectServer starts and initializes an MCP server from config.
func ConnectServer(ctx context.Context, cfg config.MCPServerConfig, envBlacklist []*regexp.Regexp) (*Server, error) {
	if err := ValidateServerName(cfg.Name); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cfg.Command) == "" {
		return nil, fmt.Errorf("mcp: server %q: command is required", cfg.Name)
	}

	env := security.SafeSubprocessEnv(cfg.Env, envBlacklist)

	c, err := mcpclient.NewStdioMCPClient(cfg.Command, env, cfg.Args...)
	if err != nil {
		return nil, fmt.Errorf("mcp: server %q: start: %w", cfg.Name, err)
	}

	initReq := mcpsdk.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcpsdk.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcpsdk.Implementation{
		Name:    "ds-code",
		Version: "0.1.0",
	}
	if _, err := c.Initialize(ctx, initReq); err != nil {
		_ = c.Close()
		return nil, fmt.Errorf("mcp: server %q: initialize: %w", cfg.Name, err)
	}

	return &Server{Name: cfg.Name, client: c}, nil
}

// ListTools returns tools advertised by the server.
func (s *Server) ListTools(ctx context.Context) ([]mcpsdk.Tool, error) {
	if s.testListTools != nil {
		return s.testListTools(ctx)
	}
	res, err := s.client.ListTools(ctx, mcpsdk.ListToolsRequest{})
	if err != nil {
		return nil, err
	}
	return res.Tools, nil
}

// CallTool invokes a tool on this server.
func (s *Server) CallTool(ctx context.Context, tool string, args json.RawMessage) (string, error) {
	if s.testCallTool != nil {
		return s.testCallTool(ctx, tool, args)
	}
	var argMap map[string]any
	if len(args) > 0 && string(args) != "null" {
		if err := json.Unmarshal(args, &argMap); err != nil {
			return "", fmt.Errorf("mcp: invalid arguments: %w", err)
		}
	}
	req := mcpsdk.CallToolRequest{}
	req.Params.Name = tool
	req.Params.Arguments = argMap

	res, err := s.client.CallTool(ctx, req)
	if err != nil {
		return "", err
	}
	return formatToolResult(res), nil
}

func formatToolResult(res *mcpsdk.CallToolResult) string {
	if res == nil {
		return ""
	}
	var parts []string
	for _, c := range res.Content {
		switch v := c.(type) {
		case mcpsdk.TextContent:
			parts = append(parts, v.Text)
		default:
			b, _ := json.Marshal(v)
			parts = append(parts, string(b))
		}
	}
	out := strings.Join(parts, "\n")
	if res.IsError && out != "" {
		return "error: " + out
	}
	return out
}

// Close shuts down the server process.
func (s *Server) Close() error {
	if s.testClose != nil {
		return s.testClose()
	}
	if s.client == nil {
		return nil
	}
	return s.client.Close()
}
