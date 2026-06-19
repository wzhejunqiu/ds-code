package read_file

const (
	DescReadFile   = "读取项目工作区内的文本文件。若你已知道需要文件的哪一部分，就只读那一部分；对大文件尤为重要。无法读取二进制或媒体文件。"
	SchemaLimitFmt = "最多读取的行数；省略时读取至文件末尾（最多 %d 行）"
	SchemaOffset   = "起始行号（从 1 开始）"

	ResultEmptyOffsetBeyond = "（空：offset %d 超出文件长度 %d）"
	MsgTruncatedMaxLines    = "\n... 已按 %d 行截断；请调整 offset/limit 继续"
	MsgMoreLinesNotShown    = "\n... 还有 %d 行未显示"
	ErrFileTooLarge         = "read_file: 文件大小 %d bytes，超过上限 %d 字节"
	ErrNotTextFile          = "read_file: 无法读取非文本文件: %s"
)
