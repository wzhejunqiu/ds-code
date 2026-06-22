package read_file

import (
	_ "embed"
	"strings"
	"text/template"

	"github.com/wzhejunqiu/ds-code/internal/tool"
)

//go:embed usage.prompt
var descTemplate string

type descVars struct {
	Shell string
}

var descTmpl = template.Must(template.New("readFile").Parse(descTemplate))

// RenderDesc returns the read_file tool description with builtin tool names injected.
func RenderDesc() string {
	var b strings.Builder
	if err := descTmpl.Execute(&b, descVars{Shell: tool.NameShell.String()}); err != nil {
		panic("read_file: desc template: " + err.Error())
	}
	return b.String()
}

const (
	SchemaFilepath      = "文件的绝对路径（不可使用相对路径）"
	ErrFilepathRequired = "filepath 为必填项"
	SchemaLimitFmt      = "最多读取的行数；省略时从文件开头读取至多 %d 行"
	SchemaOffset        = "起始行号（从 1 开始）；省略时从文件开头读取"

	ResultEmptyOffsetBeyond = "（空：offset %d 超出文件长度 %d）"
	MsgTruncatedMaxLines    = "\n... 已按 %d 行截断；请调整 offset/limit 继续"
	MsgMoreLinesNotShown    = "\n... 还有 %d 行未显示"
	ErrFileTooLarge         = "read_file: 文件大小 %d bytes，超过上限 %d 字节"
	ErrNotTextFile          = "read_file: 无法读取非文本文件: %s"
)
