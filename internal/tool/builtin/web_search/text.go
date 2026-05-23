package web_search

const (
	DescWebSearch = "搜索网页（需在配置中启用 web.search_enabled）。"

	SchemaQuery = "搜索关键词"

	ErrDisabled      = "web_search 已禁用（请设置 web.search_enabled: true）"
	ErrNotConfigured = "未配置 web_search 提供商；请对已知 URL 使用 web_fetch"
)
