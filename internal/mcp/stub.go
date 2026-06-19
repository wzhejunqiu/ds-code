package mcp

import (
	"context"

	mcpsdk "github.com/mark3labs/mcp-go/mcp"
)

// StubServer returns a Server that serves fixed tools (for unit tests).
func StubServer(name string, tools []mcpsdk.Tool) *Server {
	return &Server{
		Name: name,
		testListTools: func(context.Context) ([]mcpsdk.Tool, error) {
			return tools, nil
		},
		testClose: func() error { return nil },
	}
}
