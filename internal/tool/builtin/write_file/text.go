package write_file

import (
	_ "embed"
	"strings"
	"text/template"

	"github.com/wzhejunqiu/ds-code/internal/tool"
)

//go:embed usage.prompt
var descTemplate string

const (
	errMustReadFirstTpl      = "覆盖已有文件前须先 {{.ReadFile}} 读取以下文件：%s。请在本轮单独 {{.ReadFile}} 该文件，阅读返回内容后，再在后续回复中 {{.WriteFile}} 该文件。"
	errSameBatchReadWriteTpl = "不能在本条回复中对同一文件既 {{.ReadFile}} 又 {{.WriteFile}}：%s。{{.ReadFile}} 的返回内容仅在下一条回复中可见。请拆成两步：本轮只 {{.ReadFile}} 该文件，下一轮再 {{.WriteFile}} 该文件。"
)

type toolNameVars struct {
	ReadFile   string
	WriteFile  string
	ApplyPatch string
}

var (
	descTmpl                  = template.Must(template.New("writeFile").Parse(descTemplate))
	errMustReadFirstTmpl      = template.Must(template.New("errMustReadFirst").Parse(errMustReadFirstTpl))
	errSameBatchReadWriteTmpl = template.Must(template.New("errSameBatchReadWrite").Parse(errSameBatchReadWriteTpl))
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
		panic("write_file: template: " + err.Error())
	}
	return b.String()
}

// RenderDesc returns the write_file tool description with builtin tool names injected.
func RenderDesc() string {
	return strings.TrimSpace(renderTemplate(descTmpl, defaultToolNameVars()))
}

// ErrMustReadFirstFmt is a fmt.Sprintf template (%s = path list) with wire tool names injected.
var ErrMustReadFirstFmt = renderTemplate(errMustReadFirstTmpl, defaultToolNameVars())

// ErrSameBatchReadWriteFmt is a fmt.Sprintf template (%s = path list) with wire tool names injected.
var ErrSameBatchReadWriteFmt = renderTemplate(errSameBatchReadWriteTmpl, defaultToolNameVars())

const (
	ResultWrote = "已写入 %s（%d 字节）"
)
