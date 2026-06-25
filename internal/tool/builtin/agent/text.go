package agent

import (
	_ "embed"
	"strings"
	"text/template"

	"github.com/wzhejunqiu/ds-code/internal/tool"
)

//go:embed usage.prompt
var descTemplate string

var descTmpl = template.Must(template.New("agentUsage").Parse(descTemplate))

type descVars struct {
	Agent       string
	ReadFile    string
	Glob        string
	Grep        string
	Bash        string
	MaxParallel int
}

// RenderDesc returns the agent tool description with builtin tool names and limits injected.
func RenderDesc(maxParallel int) string {
	if maxParallel <= 0 {
		maxParallel = 3
	}
	var b strings.Builder
	if err := descTmpl.Execute(&b, descVars{
		Agent:       tool.NameAgent.String(),
		ReadFile:    tool.NameReadFile.String(),
		Glob:        tool.NameGlob.String(),
		Grep:        tool.NameGrep.String(),
		Bash:        tool.NameShell.String(),
		MaxParallel: maxParallel,
	}); err != nil {
		panic("agent: desc template: " + err.Error())
	}
	return strings.TrimSpace(b.String())
}

const (
	SchemaAgentDescription = "3–5 个词的短标题，供界面与日志展示"
	SchemaAgentPrompt      = "The task for the agent to perform"
	SchemaAgentType        = "The type of specialized agent to use for this task"
	SchemaAgentBackground  = "为 true 时在后台异步运行"

	ErrMissingParent = "agent: 缺少父会话或 tool call id"
	ErrNoStore       = "agent: 未配置 agent store"
)
