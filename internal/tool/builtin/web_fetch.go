package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
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
	if err := validateFetchURLHost(u.Hostname(), t.Cfg.Web.Allowlist); err != nil {
		return "", err
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
	dialer := &net.Dialer{Timeout: 30 * time.Second}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			if err := validateResolvedFetchHost(host); err != nil {
				return nil, err
			}
			return dialer.DialContext(ctx, network, net.JoinHostPort(host, port))
		},
	}
	return &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("web_fetch: too many redirects")
			}
			host := req.URL.Hostname()
			if err := validateFetchURLHost(host, allowlist); err != nil {
				return fmt.Errorf("redirect: %w", err)
			}
			return nil
		},
	}
}

// validateResolvedFetchHost checks host/IP at dial time (after DNS resolution).
func validateResolvedFetchHost(host string) error {
	if isBlockedFetchHost(host) {
		return fmt.Errorf("web_fetch: blocked host %q", host)
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("web_fetch: dns lookup failed for %q: %w", host, err)
	}
	for _, ip := range ips {
		if isPrivateOrMetadataIP(ip) {
			return fmt.Errorf("web_fetch: blocked IP %s for host %q", ip, host)
		}
	}
	return nil
}

