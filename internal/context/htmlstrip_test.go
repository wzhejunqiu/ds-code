package context_test

import (
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/context"
)

func TestStripHTMLTags(t *testing.T) {
	in := "<p>Hello <strong>world</strong></p>"
	got := context.StripHTMLTags(in)
	if got != "Hello world" {
		t.Fatalf("got %q", got)
	}
}

func TestStripHTMLTags_plainText(t *testing.T) {
	if context.StripHTMLTags("no tags") != "no tags" {
		t.Fatal("expected unchanged")
	}
}
