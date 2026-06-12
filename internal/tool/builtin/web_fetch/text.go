package web_fetch

const (
	DescWebFetch = "获取 URL 并返回文本内容（需启用 web.fetch_enabled 且主机在 allowlist 中）。"

	ErrDisabled         = "web_fetch 已禁用（请设置 web.fetch_enabled: true）"
	ErrInvalidURL       = "无效的 url"
	ErrSchemeNotHTTP    = "仅支持 http 与 https"
	ErrTooManyRedirects = "web_fetch: 重定向次数过多"
	ErrBlockedHost      = "web_fetch: 禁止访问的主机 %q"
	ErrDNSLookup        = "web_fetch: 解析主机 %q 失败: %w"
	ErrBlockedIP        = "web_fetch: 禁止访问的 IP %s（主机 %q）"
	ErrHostNotAllowlist = "主机 %q 不在 web.allowlist 中"
	ErrRedirectBlocked  = "redirect: %w"

	ResultHTTPPrefix = "HTTP %d\n%s"
)
