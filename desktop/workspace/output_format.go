package workspace

import (
	"context"
	"fmt"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/prompt"
	"github.com/wzhejunqiu/ds-code/internal/session"
	"github.com/wzhejunqiu/ds-code/internal/session/contentformat"
)

// ResolveAssistantOutputFormat returns the effective format for the next assistant reply.
func (m *Manager) ResolveAssistantOutputFormat(wsID, sessionID string) (string, error) {
	rt, err := m.Ensure(wsID)
	if err != nil {
		return contentformat.Markdown, err
	}
	sess, err := rt.store.Get(context.Background(), sessionID)
	if err != nil {
		return contentformat.Markdown, err
	}
	if f := contentformat.Normalize(sess.AssistantOutputFormat); f == contentformat.HTML {
		return contentformat.HTML, nil
	}
	cfg, err := config.Load(nil, config.Options{
		StartDir:           rt.app.Cfg.ProjectRoot,
		RequireAPIKey:      false,
		SkipProjectDataDir: true,
	})
	if err != nil {
		return contentformat.Markdown, nil
	}
	return contentformat.Normalize(cfg.Desktop.AssistantOutputFormat), nil
}

// GetAssistantOutputFormat returns the session-level assistant output format.
func (m *Manager) GetAssistantOutputFormat(wsID, sessionID string) (string, error) {
	rt, err := m.Ensure(wsID)
	if err != nil {
		return "", err
	}
	sess, err := rt.store.Get(context.Background(), sessionID)
	if err != nil {
		return "", err
	}
	return contentformat.Normalize(sess.AssistantOutputFormat), nil
}

// SetAssistantOutputFormat updates the session format for subsequent assistant replies.
func (m *Manager) SetAssistantOutputFormat(wsID, sessionID, format string) error {
	if format != contentformat.Markdown && format != contentformat.HTML {
		return fmt.Errorf("invalid assistant output format %q", format)
	}
	norm := contentformat.Normalize(format)
	rt, err := m.Ensure(wsID)
	if err != nil {
		return err
	}
	return rt.store.UpdateSession(context.Background(), sessionID, func(s *session.Session) error {
		s.AssistantOutputFormat = norm
		return nil
	})
}

// ApplyOutputContext configures context service overlay/format for a turn.
func (rt *Runtime) ApplyOutputContext(format string) {
	rt.ctxSvc.OutputFormat = ""
	rt.ctxSvc.OutputOverlay = ""
	if format == contentformat.HTML {
		rt.ctxSvc.OutputFormat = contentformat.HTML
		rt.ctxSvc.OutputOverlay = prompt.DesktopHTMLOutputOverlay
	}
}

// ClearOutputContext resets per-turn output overlay state.
func (rt *Runtime) ClearOutputContext() {
	rt.ctxSvc.OutputFormat = ""
	rt.ctxSvc.OutputOverlay = ""
}

// DefaultAssistantOutputFormat reads the user-level desktop default.
func DefaultAssistantOutputFormat() string {
	cfg, err := config.Load(nil, config.Options{RequireAPIKey: false, SkipProjectDataDir: true})
	if err != nil {
		return contentformat.Markdown
	}
	return contentformat.Normalize(cfg.Desktop.AssistantOutputFormat)
}
