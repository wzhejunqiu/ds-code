// Package builtin holds shared LLM-facing strings for built-in tools.
// Tool-specific strings live in each tool subpackage's text.go.
package builtin

// Schema field descriptions (shared).
const (
	SchemaPathFileRelOrAbs = "文件路径（相对或绝对）"
	SchemaPathRelRoot      = "相对项目根目录的路径"
	SchemaPathRelDefault   = "相对项目根目录的目录或文件（默认 .）"
	SchemaPathDirDefault   = "目录路径（默认 .）"
	SchemaPathsRelRoot     = "相对项目根目录的文件或目录列表"
	SchemaPattern          = "Glob 模式"
	SchemaGlobPattern      = "Glob 模式"
	SchemaFullFileContent  = "完整文件内容"
)

// Required-field and validation errors (shared).
const (
	ErrPathRequired           = "path 为必填项"
	ErrPatternRequired        = "pattern 为必填项"
	ErrPatchRequired          = "patch 为必填项"
	ErrPathsRequired          = "paths 为必填项"
	ErrQueryRequired          = "query 为必填项"
	ErrPromptRequired         = "prompt 为必填项"
	ErrOffsetLimitNonNegative = "offset 与 limit 必须为非负数"
	ErrInvalidRegex           = "无效的正则表达式"
	ErrPatternTooLong         = "pattern 过长（最多 512 字符）"
)

// DefaultMaxResults is the fallback when tools.glob.max_results is unset or non-positive.
const DefaultMaxResults = 100

// Truncation / limit suffixes (shared).
const (
	TruncatedAtEntries = "... 已截断，共 %d 项"
	TruncatedAtResults = "... 已截断，共 %d 条结果"
)
