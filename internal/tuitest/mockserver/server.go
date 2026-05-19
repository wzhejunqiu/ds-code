//go:build tuitest

package mockserver

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
)

// Server is a mock DeepSeek-compatible HTTP server.
type Server struct {
	Registry          *Registry
	HTTP              *httptest.Server
	mu                sync.Mutex
	LastAuthorization string
}

// New starts a mock LLM server backed by reg.
func New(reg *Registry) *Server {
	s := &Server{Registry: reg}
	s.HTTP = httptest.NewServer(http.HandlerFunc(s.handle))
	return s
}

// BaseURL returns the server root URL (no trailing slash).
func (s *Server) BaseURL() string {
	return s.HTTP.URL
}

// Close shuts down the server.
func (s *Server) Close() {
	s.HTTP.Close()
}

type chatBody struct {
	Stream   bool `json:"stream"`
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
}

func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || r.URL.Path != "/chat/completions" {
		http.NotFound(w, r)
		return
	}

	s.mu.Lock()
	s.LastAuthorization = r.Header.Get("Authorization")
	s.mu.Unlock()

	raw, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var body chatBody
	_ = json.Unmarshal(raw, &body)

	turn, _ := s.Registry.NextTurn()
	if turn == nil {
		turn = DefaultStopTurn("ok")
	}

	if turn.HTTPStatus > 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(turn.HTTPStatus)
		if turn.ErrBody != "" {
			_, _ = w.Write([]byte(turn.ErrBody))
		}
		return
	}

	if body.Stream {
		_ = writeStreamTurn(w, turn)
		return
	}
	_ = writeNonStreamTurn(w, turn)
}

// LastAuth returns the most recent Authorization header value.
func (s *Server) LastAuth() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.LastAuthorization
}
