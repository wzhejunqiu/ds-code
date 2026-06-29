package web_fetch

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// FetchOutcome is the result of an HTTP fetch attempt.
type FetchOutcome struct {
	Page              PageBody
	Redirected        bool
	CrossHostRedirect *url.URL
}

// fetchURL performs GET with manual redirect handling.
func fetchURL(ctx context.Context, start *url.URL, allowlist []string) (*FetchOutcome, error) {
	return fetchURLWithClient(ctx, start, allowlist, newWebFetchClient())
}

// fetchURLWithClient is like fetchURL but accepts a custom HTTP client (for tests).
func fetchURLWithClient(ctx context.Context, start *url.URL, allowlist []string, client *http.Client) (*FetchOutcome, error) {
	current := *start
	initialHost := start.Hostname()
	redirected := false

	for hops := 0; hops < 10; hops++ {
		if err := validateFetchURLHost(current.Hostname(), allowlist); err != nil {
			return nil, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, current.String(), nil)
		if err != nil {
			return nil, err
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode >= 300 && resp.StatusCode < 400 {
			loc := strings.TrimSpace(resp.Header.Get("Location"))
			_ = resp.Body.Close()
			if loc == "" {
				return nil, fmt.Errorf("%s", ErrInvalidURL)
			}
			next, err := resp.Request.URL.Parse(loc)
			if err != nil {
				return nil, fmt.Errorf("%s", ErrInvalidURL)
			}
			if !hostnamesEqual(next.Hostname(), initialHost) {
				return &FetchOutcome{CrossHostRedirect: next}, nil
			}
			redirected = true
			current = *next
			continue
		}

		body, err := io.ReadAll(io.LimitReader(resp.Body, maxFetchBodyBytes))
		_ = resp.Body.Close()
		if err != nil {
			return nil, err
		}
		return &FetchOutcome{
			Page: PageBody{
				Body:        body,
				ContentType: resp.Header.Get("Content-Type"),
				StatusCode:  resp.StatusCode,
			},
			Redirected: redirected,
		}, nil
	}
	return nil, fmt.Errorf("%s", ErrTooManyRedirects)
}

// PageToMarkdown converts a fetched page body to markdown when appropriate.
func PageToMarkdown(page PageBody) string {
	if !looksLikeHTML(page.Body, page.ContentType) {
		return string(page.Body)
	}
	out, err := htmlToMarkdown(string(page.Body))
	if err != nil || strings.TrimSpace(out) == "" {
		return string(page.Body)
	}
	return out
}

func looksLikeHTML(body []byte, contentType string) bool {
	if strings.Contains(strings.ToLower(contentType), "text/html") {
		return true
	}
	trim := bytes.TrimSpace(body)
	if len(trim) == 0 {
		return false
	}
	lower := strings.ToLower(string(trim))
	return strings.HasPrefix(lower, "<!doctype") || strings.HasPrefix(lower, "<html")
}
