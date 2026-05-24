package mock

import (
	"context"
	"sync"

	"github.com/wzhejunqiu/ds-code/internal/llm"
)

// Client is a test double that returns scripted responses per call.
type Client struct {
	mu        sync.Mutex
	Responses []*llm.Response
	Errors    []error
	Calls     []llm.Request
}

func (m *Client) Chat(ctx context.Context, req llm.Request) (*llm.Response, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, req)
	if len(m.Errors) > 0 {
		err := m.Errors[0]
		m.Errors = m.Errors[1:]
		return nil, err
	}
	if len(m.Responses) == 0 {
		return &llm.Response{Content: "done", FinishReason: "stop"}, nil
	}
	resp := m.Responses[0]
	m.Responses = m.Responses[1:]
	return resp, nil
}
