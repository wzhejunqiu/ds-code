package read_file

const (
	DescReadFile = `读取本地文件系统中的文件。你可通过该工具直接访问任意文件。假定本工具能够读取设备上的所有文件。若用户提供文件路径，则默认该路径有效。读取不存在的文件也属正常操作，此时工具会返回错误信息。

	用法：
	- 本工具仅支持读取文件，无法读取目录。如需读取目录内容，请通过 %s 工具执行 ls 命令。
	- 若你已知道需要文件的哪一部分，就只读那一部分；对大文件尤为重要。
	- 无法读取二进制或媒体文件。
	`
	SchemaLimitFmt = "最多读取的行数；省略时读取至文件末尾（最多 %d 行）"
	SchemaOffset   = "起始行号（从 1 开始）"

	ResultEmptyOffsetBeyond = "（空：offset %d 超出文件长度 %d）"
	MsgTruncatedMaxLines    = "\n... 已按 %d 行截断；请调整 offset/limit 继续"
	MsgMoreLinesNotShown    = "\n... 还有 %d 行未显示"
	ErrFileTooLarge         = "read_file: 文件大小 %d bytes，超过上限 %d 字节"
	ErrNotTextFile          = "read_file: 无法读取非文本文件: %s"
)
