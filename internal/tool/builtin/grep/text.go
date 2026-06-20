package grep

const (
	DescGrep = "在工作区内用正则搜索文件内容；始终跳过 .git，可选 tools.search.skip_dirs；搜索前应收窄 path 避免盲目全库扫描。凡内容搜索任务必须调用本 grep 工具，禁止通过 bash 工具执行 grep 或 rg。"

	SchemaRegexPattern = "正则表达式（匹配文件行内容）"
	SchemaGrepPath     = "相对项目根的路径：目录、单文件或 glob（如 *.go、internal/**/*.go），默认 ."
	SchemaOutputMode   = "输出模式：content=匹配行 path:行号:内容；files_with_matches=仅文件路径（默认）；count=总匹配数"
)
