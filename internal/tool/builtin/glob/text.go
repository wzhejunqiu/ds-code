package glob

import (
	_ "embed"
	"strings"
	"text/template"

	"github.com/wzhejunqiu/ds-code/internal/tool"
)

//go:embed usage.prompt
var descTemplate string

var descTmpl = template.Must(template.New("globUsage").Parse(descTemplate))

type descVars struct {
	Glob     string
	Bash     string
	ReadFile string
	Grep     string
	Agent    string
}

// RenderDesc returns the glob tool description.
func RenderDesc() string {
	var b strings.Builder
	if err := descTmpl.Execute(&b, descVars{
		Glob:     tool.NameGlob.String(),
		Bash:     tool.NameShell.String(),
		ReadFile: tool.NameReadFile.String(),
		Grep:     tool.NameGrep.String(),
		Agent:    tool.NameAgent.String(),
	}); err != nil {
		panic("glob: desc template: " + err.Error())
	}
	return strings.TrimSpace(b.String())
}

const (
	SchemaPattern = "Glob 模式（例如 \"**/*.go\"、\"*_test.go\"），对应 rg --glob"
	SchemaPath    = "搜索根目录（必须是目录，rg PATH）。默认为当前工作目录（.）。"

	MsgRipgrepTimeout   = "glob 搜索超时"
	MsgResultsTruncated = "（结果已截断，请使用更具体的 path 或 pattern）"
)
