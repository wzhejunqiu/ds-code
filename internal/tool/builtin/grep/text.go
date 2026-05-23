package grep

const (
	DescGrep = "在工作区内用正则搜索文件内容，遵循 .gitignore 与权限策略。凡内容搜索任务必须调用本 grep 工具，禁止通过 shell 执行 grep 或 rg。path 可为目录/文件或 glob（如 internal/**/*.go）。结果按命中文件的最近修改时间排序（越新越靠前）。"

	SchemaRegexPattern = "正则表达式（匹配文件行内容）"
	SchemaGrepPath     = "相对项目根的路径：目录、单文件或 glob（如 *.go、internal/**/*.go），默认 ."
)
