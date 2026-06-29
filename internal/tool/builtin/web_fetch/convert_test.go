package web_fetch_test

import (
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/tool/builtin/web_fetch"
)

func TestPageToMarkdown_html(t *testing.T) {
	page := web_fetch.PageBody{
		Body:        []byte("<html><body><h1>Title</h1><p>text</p></body></html>"),
		ContentType: "text/html",
	}
	out := web_fetch.PageToMarkdown(page)
	if !strings.Contains(out, "Title") {
		t.Fatalf("out = %q", out)
	}
}

func TestPageToMarkdown_plain(t *testing.T) {
	page := web_fetch.PageBody{Body: []byte(`{"ok":true}`), ContentType: "application/json"}
	out := web_fetch.PageToMarkdown(page)
	if out != `{"ok":true}` {
		t.Fatalf("out = %q", out)
	}
}
