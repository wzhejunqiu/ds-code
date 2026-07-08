package agent

import (
	"github.com/wzhejunqiu/ds-code/internal/session"
	"github.com/wzhejunqiu/ds-code/internal/session/contentformat"
)

func applyOutputFormat(msg *session.Message, format string) {
	if format == contentformat.HTML {
		msg.ContentFormat = contentformat.HTML
	}
}

func outputFormatFromRunner(r *Runner) string {
	if r == nil || r.Context == nil {
		return ""
	}
	return r.Context.OutputFormat
}
