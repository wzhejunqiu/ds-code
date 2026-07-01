package web_fetch_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/llm/mock"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/permissionmode"
	"github.com/wzhejunqiu/ds-code/internal/tool/builtin/web_fetch"
)

const testFetchHost = "example.com"

func testPermAuto() *permission.Engine {
	e := permission.NewEngine(permissionmode.Auto, "/tmp", false)
	return e
}

func testStartURL(path string) *url.URL {
	u, _ := url.Parse("http://" + testFetchHost + path)
	return u
}

func TestFetchURL_crossHostRedirect(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://other.test/page", http.StatusFound)
	}))
	defer srv.Close()

	client := web_fetch.TestFetchClient(srv.URL)
	ctx := permission.WithWebFetchApproval(context.Background())
	out, err := web_fetch.FetchURLWithClient(ctx, testStartURL("/"), testPermAuto(), client)
	if err != nil {
		t.Fatal(err)
	}
	if out.CrossHostRedirect == nil || out.CrossHostRedirect.Host != "other.test" {
		t.Fatalf("redirect = %v", out.CrossHostRedirect)
	}
}

func TestFetchURL_sameDomainRedirectNotMarkedForCache(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/start" {
			http.Redirect(w, r, "/final", http.StatusFound)
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	client := web_fetch.TestFetchClient(srv.URL)
	ctx := permission.WithWebFetchApproval(context.Background())
	out, err := web_fetch.FetchURLWithClient(ctx, testStartURL("/start"), testPermAuto(), client)
	if err != nil {
		t.Fatal(err)
	}
	if !out.Redirected {
		t.Fatal("expected redirected=true")
	}
	if string(out.Page.Body) != "ok" {
		t.Fatalf("body = %q", out.Page.Body)
	}
}

func TestWebFetch_cacheSkipsHTTPOnSecondCall(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		_, _ = w.Write([]byte("plain text body"))
	}))
	defer srv.Close()

	cfg := &config.Config{
		Web: config.WebConfig{FetchEnabled: true, Allowlist: []string{testFetchHost}},
		LLM: config.LLMConfig{MaxTokens: 4096},
	}
	llmMock := &mock.Client{
		Responses: []*llm.Response{
			{Content: "a1", FinishReason: "stop"},
			{Content: "a2", FinishReason: "stop"},
		},
	}
	cache := web_fetch.NewLRUCache(cfg.Web)
	tool := &web_fetch.WebFetchTool{
		Cfg:    cfg,
		Perm:   testPermAuto(),
		Strict: false,
		LLM:    llmMock,
		Cache:  cache,
	}

	cache.Put("https://"+testFetchHost+"/", web_fetch.PageBody{
		Body:        []byte("plain text body"),
		ContentType: "text/plain",
		StatusCode:  200,
	})

	args := `{"url":"http://` + testFetchHost + `/","prompt":"p"}`
	if _, err := tool.Execute(context.Background(), json.RawMessage(args)); err != nil {
		t.Fatal(err)
	}
	if _, err := tool.Execute(context.Background(), json.RawMessage(args)); err != nil {
		t.Fatal(err)
	}
	if hits.Load() != 0 {
		t.Fatalf("hits = %d, want 0", hits.Load())
	}
	if len(llmMock.Calls) != 2 {
		t.Fatalf("llm calls = %d, want 2", len(llmMock.Calls))
	}
}

func TestWebFetch_requiresPrompt(t *testing.T) {
	cfg := &config.Config{Web: config.WebConfig{FetchEnabled: true, Allowlist: []string{"example.com"}}}
	tool := &web_fetch.WebFetchTool{Cfg: cfg, Perm: testPermAuto(), LLM: &mock.Client{}}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"url":"https://example.com"}`))
	if err == nil || !strings.Contains(err.Error(), "prompt") {
		t.Fatalf("err = %v", err)
	}
}

func TestFetchURL_sameHostRedirectWithOnceApproval(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/start" {
			http.Redirect(w, r, "/final", http.StatusFound)
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	perm := permission.NewEngine(permissionmode.Readonly, t.TempDir(), true)
	perm.WebFetchPrompter = func(host, rawURL string) (permission.WebFetchChoice, error) {
		return permission.WebFetchAllowOnce, nil
	}
	ctx, err := perm.PrepareWebFetch(context.Background(), map[string]any{"url": "http://" + testFetchHost + "/start"})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}

	client := web_fetch.TestFetchClient(srv.URL)
	out, err := web_fetch.FetchURLWithClient(ctx, testStartURL("/start"), perm, client)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if !out.Redirected {
		t.Fatal("expected redirected=true")
	}
	if string(out.Page.Body) != "ok" {
		t.Fatalf("body = %q", out.Page.Body)
	}
}
