package web_fetch

import (
	"net/url"
	"strings"
)

const maxFetchBodyBytes = 512 * 1024

// normalizeURL canonicalizes a URL for cache keys and requests (http→https, drop fragment, lower host).
func normalizeURL(raw string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return nil, err
	}
	if u.Scheme == "http" && (u.Port() == "" || u.Port() == "80") {
		u.Scheme = "https"
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return nil, err
	}
	u.Fragment = ""
	u.Host = strings.ToLower(u.Host)
	return u, nil
}

func cacheKey(u *url.URL) string {
	return u.String()
}

func hostnamesEqual(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

func formatCrossHostRedirect(target *url.URL) string {
	return "REDIRECT: " + target.String() + "\n该 URL 已跨域名重定向。请使用上述 URL 重新发起 web_fetch 请求以获取页面内容。"
}
