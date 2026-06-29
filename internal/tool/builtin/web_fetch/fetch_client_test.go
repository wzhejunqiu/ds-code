package web_fetch

import (
	"net/http"
	"testing"
)

func TestNewWebFetchClient_noAutoRedirect(t *testing.T) {
	client := newWebFetchClient()
	first, err := http.NewRequest(http.MethodGet, "http://allowed.test/start", nil)
	if err != nil {
		t.Fatal(err)
	}
	next, err := http.NewRequest(http.MethodGet, "http://allowed.test/next", nil)
	if err != nil {
		t.Fatal(err)
	}
	err = client.CheckRedirect(next, []*http.Request{first})
	if err != http.ErrUseLastResponse {
		t.Fatalf("CheckRedirect err = %v", err)
	}
}
