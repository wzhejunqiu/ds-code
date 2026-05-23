package read_file

const (
	DescReadFile = "读取项目工作区内的文件，支持 offset（从 1 开始）与 limit（单次最多行数）。若你已知道需要文件的哪一部分，就只读那一部分；对大文件尤为重要。"

	ResultEmptyOffsetBeyond = "（空：offset %d 超出文件长度 %d）"
	MsgTruncatedMaxLines      = "\n... 已按 max_lines 截断；请调整 offset/limit 继续"
	MsgMoreLinesNotShown      = "\n... 还有 %d 行未显示"
	ErrFileTooLarge           = "read_file: 文件大小 %d 超过上限 %d 字节"
)
