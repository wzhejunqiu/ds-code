package apply_patch

import (
	_ "embed"
	"strings"
	"text/template"

	"github.com/wzhejunqiu/ds-code/internal/tool"
)

//go:embed usage.prompt
var descTemplate string

const (
	errMustReadFirstTpl     = "编辑前须先 {{.ReadFile}} 读取以下文件：%s。请在本轮单独 {{.ReadFile}} 该文件，阅读返回内容后，再在后续回复中 {{.ApplyPatch}} 该文件。"
	errSameBatchReadEditTpl = "不能在本条回复中对同一文件既 {{.ReadFile}} 又 {{.ApplyPatch}}：%s。{{.ReadFile}} 的返回内容仅在下一条回复中可见。请拆成两步：本轮只 {{.ReadFile}} 该文件，下一轮再 {{.ApplyPatch}} 该文件。"
)

type toolNameVars struct {
	ReadFile   string
	WriteFile  string
	ApplyPatch string
}

var (
	descTmpl                 = template.Must(template.New("applyPatch").Parse(descTemplate))
	errMustReadFirstTmpl     = template.Must(template.New("errMustReadFirst").Parse(errMustReadFirstTpl))
	errSameBatchReadEditTmpl = template.Must(template.New("errSameBatchReadEdit").Parse(errSameBatchReadEditTpl))
)

func defaultToolNameVars() toolNameVars {
	return toolNameVars{
		ReadFile:   tool.NameReadFile.String(),
		WriteFile:  tool.NameWriteFile.String(),
		ApplyPatch: tool.NameApplyPatch.String(),
	}
}

func renderTemplate(t *template.Template, vars toolNameVars) string {
	var b strings.Builder
	if err := t.Execute(&b, vars); err != nil {
		panic("apply_patch: template: " + err.Error())
	}
	return b.String()
}

// RenderDesc returns the apply_patch tool description with builtin tool names injected.
func RenderDesc() string {
	return renderTemplate(descTmpl, defaultToolNameVars())
}

// ErrMustReadFirstFmt is a fmt.Sprintf template (%s = path list) with wire tool names injected.
var ErrMustReadFirstFmt = renderTemplate(errMustReadFirstTmpl, defaultToolNameVars())

// ErrSameBatchReadEditFmt is a fmt.Sprintf template (%s = path list) with wire tool names injected.
var ErrSameBatchReadEditFmt = renderTemplate(errSameBatchReadEditTmpl, defaultToolNameVars())

const (
	SchemaPatchBody = "完整的 Codex 风格的补丁文本"

	ResultAppliedPrefix = "已应用："
)
