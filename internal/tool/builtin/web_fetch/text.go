package web_fetch

import (
	_ "embed"
	"strings"
	"text/template"

	"github.com/wzhejunqiu/ds-code/internal/tool"
)

//go:embed usage.prompt
var descTemplate string

var descTmpl = template.Must(template.New("webFetchUsage").Parse(descTemplate))

type descVars struct {
	WebFetch string
	Bash     string
}

// RenderDesc returns the web_fetch tool description with builtin tool names injected.
func RenderDesc() string {
	var b strings.Builder
	if err := descTmpl.Execute(&b, descVars{
		WebFetch: tool.NameWebFetch.String(),
		Bash:     tool.NameShell.String(),
	}); err != nil {
		panic("web_fetch: desc template: " + err.Error())
	}
	return strings.TrimSpace(b.String())
}

const (
	SchemaURL    = "要获取的 HTTP 或 HTTPS URL（须为完整合法 URL；http 将自动升级为 https）"
	SchemaPrompt = "对抓取内容执行的提示词"

	ErrDisabled         = "web_fetch 已禁用（请设置 web.fetch_enabled: true）"
	ErrPromptRequired   = "prompt 为必填项"
	ErrInvalidURL       = "无效的 url"
	ErrSchemeNotHTTP    = "仅支持 http 与 https"
	ErrTooManyRedirects = "web_fetch: 重定向次数过多"
	ErrBlockedHost      = "web_fetch: 禁止访问的主机 %q"
	ErrDNSLookup        = "web_fetch: 解析主机 %q 失败: %w"
	ErrBlockedIP        = "web_fetch: 禁止访问的 IP %s（主机 %q）"
	ErrHostNotAllowlist = "主机 %q 不在 web.allowlist 中"
	ErrRedirectBlocked  = "redirect: %w"
)
