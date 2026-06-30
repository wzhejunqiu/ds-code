package permission

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/permissionmode"
)

// WebFetchChoice is the user's response to an out-of-allowlist web_fetch host.
type WebFetchChoice int

const (
	WebFetchDeny WebFetchChoice = iota
	WebFetchAllowOnce
	WebFetchAllowAlways
)

// WebFetchPrompter asks the user to approve a web_fetch host not in allowlist.
type WebFetchPrompter func(host, rawURL string) (WebFetchChoice, error)

type webFetchCtxKey struct{}

type webFetchApproval struct {
	onceHosts map[string]struct{}
}

// WithWebFetchApproval returns ctx carrying once-approval state for one tool invocation.
func WithWebFetchApproval(ctx context.Context) context.Context {
	if approvalFromCtx(ctx) != nil {
		return ctx
	}
	return context.WithValue(ctx, webFetchCtxKey{}, &webFetchApproval{
		onceHosts: make(map[string]struct{}),
	})
}

func approvalFromCtx(ctx context.Context) *webFetchApproval {
	if ctx == nil {
		return nil
	}
	v, _ := ctx.Value(webFetchCtxKey{}).(*webFetchApproval)
	return v
}

func isOnceApproved(ctx context.Context, host string) bool {
	a := approvalFromCtx(ctx)
	if a == nil {
		return false
	}
	_, ok := a.onceHosts[normalizeFetchHost(host)]
	return ok
}

func approveOnce(ctx context.Context, host string) context.Context {
	host = normalizeFetchHost(host)
	if host == "" {
		return ctx
	}
	a := approvalFromCtx(ctx)
	if a == nil {
		ctx = WithWebFetchApproval(ctx)
		a = approvalFromCtx(ctx)
	}
	a.onceHosts[host] = struct{}{}
	return ctx
}

func hostFromURL(raw string) (host string, err error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return "", fmt.Errorf("%w: invalid web_fetch url", ErrDenied)
	}
	return u.Hostname(), nil
}

// PrepareWebFetch validates URL/host policy and prompts when needed (readonly/ask).
func (e *Engine) PrepareWebFetch(ctx context.Context, args map[string]any) (context.Context, error) {
	rawURL, _ := args["url"].(string)
	ctx = WithWebFetchApproval(ctx)
	host, err := hostFromURL(rawURL)
	if err != nil {
		return ctx, err
	}
	return e.authorizeFetchHost(ctx, host, rawURL)
}

// CheckFetchHost applies SSRF + mode policy for a redirect hop.
func (e *Engine) CheckFetchHost(ctx context.Context, host string) error {
	if err := CheckFetchSSRF(host); err != nil {
		return err
	}
	if e.Mode == permissionmode.Auto {
		return nil
	}
	if e.hostInAllowlist(host) || isOnceApproved(ctx, host) {
		return nil
	}
	return fmt.Errorf(errHostNotAllowlist, host)
}

func (e *Engine) authorizeFetchHost(ctx context.Context, host, rawURL string) (context.Context, error) {
	if err := CheckFetchSSRF(host); err != nil {
		return ctx, err
	}
	if e.Mode == permissionmode.Auto {
		return ctx, nil
	}
	if e.hostInAllowlist(host) || isOnceApproved(ctx, host) {
		return ctx, nil
	}
	if !e.Interactive || e.WebFetchPrompter == nil {
		return ctx, ErrNeedTTY
	}
	choice, err := e.WebFetchPrompter(host, rawURL)
	if err != nil {
		return ctx, err
	}
	switch choice {
	case WebFetchAllowOnce:
		return approveOnce(ctx, host), nil
	case WebFetchAllowAlways:
		if e.ProjectRoot == "" {
			return ctx, fmt.Errorf("%w: no project root for allowlist persist", ErrDenied)
		}
		if err := config.AppendWebAllowlist(e.ProjectRoot, host); err != nil {
			return ctx, fmt.Errorf("%w: persist allowlist: %v", ErrDenied, err)
		}
		e.WebAllowlist = appendUniqueAllowlist(e.WebAllowlist, host)
		return ctx, nil
	case WebFetchDeny:
		return ctx, ErrRejected
	default:
		return ctx, ErrRejected
	}
}
