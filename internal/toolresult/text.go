package toolresult

// ToolErrorPrefix prefixes formatted tool errors (see UnpackToolBody).
const ToolErrorPrefix = "错误: "

// TruncateSuffix is appended when tool output exceeds the configured limit.
const TruncateSuffix = "\n... [已截断；请使用 offset/limit 或缩小查询范围]"
