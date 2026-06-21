package shell

import (
	_ "embed"
	"strings"
	"text/template"

	"github.com/wzhejunqiu/ds-code/internal/tool"
)

//go:embed usage.prompt
var descTemplate string

//go:embed shell_cmd_description.prompt
var SchemaShellDescription string

// descVars holds named values injected into the bash tool description template.
type descVars struct {
	Bash       string
	ReadFile   string
	Grep       string
	Glob       string
	ListDir    string
	ApplyPatch string
	WriteFile  string
}

var descTmpl = template.Must(template.New("bashUsage").Parse(descTemplate))

// RenderDesc returns the bash tool description with builtin tool names injected.
func RenderDesc() string {
	return renderDesc(defaultDescVars())
}

func defaultDescVars() descVars {
	return descVars{
		Bash:       tool.NameShell.String(),
		ReadFile:   tool.NameReadFile.String(),
		Grep:       tool.NameGrep.String(),
		Glob:       tool.NameGlob.String(),
		ListDir:    tool.NameListDir.String(),
		ApplyPatch: tool.NameApplyPatch.String(),
		WriteFile:  tool.NameWriteFile.String(),
	}
}

func renderDesc(vars descVars) string {
	var b strings.Builder
	if err := descTmpl.Execute(&b, vars); err != nil {
		panic("shell: bash desc template: " + err.Error())
	}
	return b.String()
}

const (
	SchemaShellCommand    = "要执行的 bash 命令字符串"
	SchemaRunInBackground = "为 true 时在后台并行执行并阻塞至完成；同轮可与其他 run_in_background 或只读工具并行；不要在 command 末尾加 &"
	SchemaTimeoutMs       = "可选，command 的超时毫秒数；省略时使用 tools.shell.timeout（默认 120 秒）；最大 600000；超时将强制终止子进程"

	ErrBackgroundUnavailable = "shell 后台任务不可用"
	ErrCommandRequired       = "command 为必填项"
	ErrCommandRequiredSync   = "command 为必填项"
	ErrTimeoutMsNonNegative  = "timeout_ms 必须为非负整数"

	ResultNoOutput   = "（无输出）"
	ResultExitPrefix = "exit: "
)
