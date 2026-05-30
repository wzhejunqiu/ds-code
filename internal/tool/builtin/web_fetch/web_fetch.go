package web_fetch

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

	ctxpkg "github.com/wzhejunqiu/ds-code/internal/context"
	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/tool"
	"github.com/wzhejunqiu/ds-code/internal/tool/builtin"
)

// WebFetchTool fetches a URL when enabled and allowlisted.
type WebFetchTool struct {
	Cfg    *config.Config
	Strict bool
}

func (t *WebFetchTool) Name() string { return tool.NameWebFetch.String() }

func (t *WebFetchTool) IsReadOnly() bool        { return true }
func (t *WebFetchTool) IsConcurrencySafe() bool { return true }

func (t *WebFetchTool) Description() string { return DescWebFetch }

func (t *WebFetchTool) Schema() map[string]any {
	return tool.ObjectSchema(map[string]any{
		"url": map[string]any{"type": "string", "description": builtin.SchemaHTTPURL},
	}, []string{"url"}, t.Strict)
}

func (t *WebFetchTool) PermissionLevel() permission.Level { return permission.LevelMedium }

func (t *WebFetchTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	if !t.Cfg.Web.FetchEnabled {
		return "", fmt.Errorf("%s", ErrDisabled)
	}
	var in struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", err
	}
	u, err := url.Parse(strings.TrimSpace(in.URL))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("%s", ErrInvalidURL)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("%s", ErrSchemeNotHTTP)
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
	out := fmt.Sprintf(ResultHTTPPrefix, resp.StatusCode, string(body))
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
				return fmt.Errorf("%s", ErrTooManyRedirects)
			}
			host := req.URL.Hostname()
			if err := validateFetchURLHost(host, allowlist); err != nil {
				return fmt.Errorf(ErrRedirectBlocked, err)
			}
			return nil
		},
	}
}

func validateResolvedFetchHost(host string) error {
	if isBlockedFetchHost(host) {
		return fmt.Errorf(ErrBlockedHost, host)
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf(ErrDNSLookup, host, err)
	}
	for _, ip := range ips {
		if isPrivateOrMetadataIP(ip) {
			return fmt.Errorf(ErrBlockedIP, ip, host)
		}
	}
	return nil
}
