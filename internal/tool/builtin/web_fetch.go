package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	ctxpkg "github.com/hejunqiu/ds-code/internal/context"
	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/permission"
	"github.com/hejunqiu/ds-code/internal/tool"
)

// WebFetchTool fetches a URL when enabled and allowlisted.
type WebFetchTool struct {
	Cfg    *config.Config
	Strict bool
}

func (t *WebFetchTool) Name() string { return "web_fetch" }

func (t *WebFetchTool) Description() string {
	return "Fetch a URL and return text content (requires web.fetch_enabled and host allowlist)."
}

func (t *WebFetchTool) Schema() map[string]any {
	return tool.ObjectSchema(map[string]any{
		"url": map[string]any{"type": "string", "description": "HTTP or HTTPS URL"},
	}, []string{"url"}, t.Strict)
}

func (t *WebFetchTool) PermissionLevel() permission.Level { return permission.LevelMedium }

func (t *WebFetchTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	if !t.Cfg.Web.FetchEnabled {
		return "", fmt.Errorf("web_fetch is disabled (set web.fetch_enabled: true)")
	}
	var in struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", err
	}
	u, err := url.Parse(strings.TrimSpace(in.URL))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("invalid url")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("only http and https are supported")
	}
	if !hostAllowed(u.Hostname(), t.Cfg.Web.Allowlist) {
		return "", fmt.Errorf("host %q is not in web.allowlist", u.Hostname())
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", err
	}
	client := newWebFetchClient(t.Cfg.Web.Allowlist)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return "", err
	}
	out := fmt.Sprintf("HTTP %d\n%s", resp.StatusCode, string(body))
	return ctxpkg.TruncateToolResult(out, t.Cfg), nil
}

func newWebFetchClient(allowlist []string) *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("web_fetch: too many redirects")
			}
			host := req.URL.Hostname()
			if !hostAllowed(host, allowlist) {
				return fmt.Errorf("redirect host %q is not in web.allowlist", host)
			}
			return nil
		},
	}
}

func hostAllowed(host string, allowlist []string) bool {
	if len(allowlist) == 0 {
		return false
	}
	host = strings.ToLower(host)
	for _, entry := range allowlist {
		entry = strings.ToLower(strings.TrimSpace(entry))
		if entry == "" {
			continue
		}
		if entry == host {
			return true
		}
		if strings.HasPrefix(entry, "*.") {
			suffix := entry[1:]
			if strings.HasSuffix(host, suffix) || host == strings.TrimPrefix(entry, "*") {
				return true
			}
		}
	}
	return false
}
