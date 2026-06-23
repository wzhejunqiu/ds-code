package grep

import (
	_ "embed"
	"strings"
	"text/template"

	"github.com/wzhejunqiu/ds-code/internal/tool"
)

//go:embed usage.prompt
var descTemplate string

var descTmpl = template.Must(template.New("grepUsage").Parse(descTemplate))

type descVars struct {
	Grep  string
	Bash  string
	Agent string
}

// RenderDesc returns the grep tool description.
func RenderDesc() string {
	var b strings.Builder
	if err := descTmpl.Execute(&b, descVars{
		Grep:  tool.NameGrep.String(),
		Bash:  tool.NameShell.String(),
		Agent: tool.NameAgent.String(),
	}); err != nil {
		panic("grep: desc template: " + err.Error())
	}
	return strings.TrimSpace(b.String())
}

const (
	SchemaPattern     = "在文件内容中搜索的正则表达式（ripgrep / Rust regex 语法）"
	SchemaPath        = "要搜索的文件或目录（rg PATH）。默认为当前工作目录。"
	SchemaGlob        = "过滤文件的 glob 模式（例如 \"*.js\"、\"*.{ts,tsx}\"），对应 rg --glob"
	SchemaOutputMode  = "输出模式：\"content\" 显示匹配行；\"files_with_matches\" 显示文件路径（默认）；\"count\" 显示匹配计数"
	SchemaBefore      = "每个匹配前显示的上下文行数（rg -B）。需要 output_mode: \"content\"。"
	SchemaAfter       = "每个匹配后显示的上下文行数（rg -A）。需要 output_mode: \"content\"。"
	SchemaContextC    = "context 的别名（rg -C）。需要 output_mode: \"content\"。"
	SchemaContext     = "每个匹配前后显示的上下文行数（rg -C）。需要 output_mode: \"content\"。"
	SchemaLineNumbers = "在输出中显示行号（rg -n）。需要 output_mode: \"content\"。默认为 true。"
	SchemaIgnoreCase  = "忽略大小写搜索（rg -i）"
	SchemaFileType    = "要搜索的文件类型（rg --type）。常见类型：js、py、rust、go、java 等。"
	SchemaHeadLimit   = "限制输出为前 N 行/条；三种 output_mode 均适用。未传时默认 tools.grep.head_limit（250）；传 0 表示不限。"
	SchemaOffset      = "在应用 head_limit 前先跳过前 N 行/条。默认为 0。"
	SchemaMultiline   = "启用多行模式（rg -U --multiline-dotall）。默认：false。"

	MsgRipgrepTimeout = "grep 搜索超时"
)
