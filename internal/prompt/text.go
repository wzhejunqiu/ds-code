package prompt

import (
	_ "embed"
	"strings"
	"text/template"

	"github.com/wzhejunqiu/ds-code/internal/tool"
	appver "github.com/wzhejunqiu/ds-code/internal/version"
)

//go:embed prompt.md
var defaultSystemBaseTemplate string

// systemBaseVars holds named values injected into the default system prompt template.
type systemBaseVars struct {
	AppName    string
	Bash       string
	ReadFile   string
	ApplyPatch string
	WriteFile  string
	Glob       string
	Grep       string
}

var defaultSystemBaseTmpl = template.Must(template.New("defaultSystemBase").Parse(defaultSystemBaseTemplate))

// DefaultSystemBase returns the built-in system prompt with builtin tool names injected.
func DefaultSystemBase() string {
	return renderSystemBase(defaultSystemBaseVars())
}

func defaultSystemBaseVars() systemBaseVars {
	return systemBaseVars{
		AppName:    appver.Name,
		Bash:       tool.NameShell.String(),
		ReadFile:   tool.NameReadFile.String(),
		ApplyPatch: tool.NameApplyPatch.String(),
		WriteFile:  tool.NameWriteFile.String(),
		Glob:       tool.NameGlob.String(),
		Grep:       tool.NameGrep.String(),
	}
}

func renderSystemBase(vars systemBaseVars) string {
	var b strings.Builder
	if err := defaultSystemBaseTmpl.Execute(&b, vars); err != nil {
		panic("prompt: default system base template: " + err.Error())
	}
	return b.String()
}

// MergeSystem section headers.
const (
	SectionRuntimeEnv    = "\n\n## 运行环境\n"
	SectionAgentsMD      = "\n\n## 项目说明（AGENTS.md）\n"
	SectionRules         = "\n\n## 规则\n"
	SectionSkill         = "\n\n## 当前 Skill\n"
	SectionGit           = "\n\n## Git 快照\n\n以下为对话开始时刻的仓库状态快照，不会随对话进行而自动更新。\n\n"
	SectionAgentOverlay  = "\n\n<agent-type-overlay>\n"
	SectionOutputOverlay = "\n\n## 桌面输出格式\n"
	SectionTools         = "\n\n## 工具定义（静态）\n"
)
