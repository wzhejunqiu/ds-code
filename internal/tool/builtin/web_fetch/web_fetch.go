package web_fetch

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/config"
	ctxpkg "github.com/wzhejunqiu/ds-code/internal/context"
	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/tool"
)

var (
	sharedCache   *LRUCache
	sharedCacheMu sync.Mutex
)

// WebFetchTool fetches a URL, converts HTML to Markdown, and analyzes with a lightweight model.
type WebFetchTool struct {
	Cfg    *config.Config
	Perm   *permission.Engine
	Strict bool
	LLM    llm.Client
	Cache  *LRUCache
}

func (t *WebFetchTool) Name() string { return tool.NameWebFetch.String() }

func (t *WebFetchTool) WithPerm(perm *permission.Engine) tool.Tool {
	out := *t
	out.Perm = perm
	return &out
}

func (t *WebFetchTool) IsReadOnly() bool        { return true }
func (t *WebFetchTool) IsConcurrencySafe() bool { return true }

func (t *WebFetchTool) Description() string { return RenderDesc() }

func (t *WebFetchTool) Schema() map[string]any {
	return tool.ObjectSchema(map[string]any{
		"url":    map[string]any{"type": "string", "description": SchemaURL},
		"prompt": map[string]any{"type": "string", "description": SchemaPrompt},
	}, []string{"url", "prompt"}, t.Strict)
}

func (t *WebFetchTool) PermissionLevel() permission.Level { return permission.LevelMedium }

func (t *WebFetchTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	if !t.Cfg.Web.FetchEnabled {
		return "", fmt.Errorf("%s", ErrDisabled)
	}
	var in struct {
		URL    string `json:"url"`
		Prompt string `json:"prompt"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", err
	}
	if strings.TrimSpace(in.Prompt) == "" {
		return "", fmt.Errorf("%s", ErrPromptRequired)
	}

	u, err := normalizeURL(in.URL)
	if err != nil {
		return "", fmt.Errorf("%s", ErrInvalidURL)
	}
	key := cacheKey(u)
	cache := t.cache()

	var page PageBody
	if cached := cache.Get(key); cached != nil {
		page = *cached
	} else {
		outcome, err := fetchURL(ctx, u, t.Perm, nil)
		if err != nil {
			return "", err
		}
		if outcome.CrossHostRedirect != nil {
			return formatCrossHostRedirect(outcome.CrossHostRedirect), nil
		}
		page = outcome.Page
		if !outcome.Redirected {
			cache.Put(key, page)
		}
	}

	markdown := PageToMarkdown(page)
	result, err := AnalyzePage(ctx, t.LLM, t.Cfg, in.Prompt, markdown)
	if err != nil {
		return "", err
	}
	return ctxpkg.TruncateToolResult(result, t.Cfg), nil
}

func (t *WebFetchTool) cache() *LRUCache {
	if t.Cache != nil {
		return t.Cache
	}
	sharedCacheMu.Lock()
	defer sharedCacheMu.Unlock()
	if sharedCache == nil {
		sharedCache = NewLRUCache(t.Cfg.Web)
	}
	return sharedCache
}

// SharedCache returns a process-wide LRU cache for web_fetch (used by setup).
func SharedCache(web config.WebConfig) *LRUCache {
	sharedCacheMu.Lock()
	defer sharedCacheMu.Unlock()
	if sharedCache == nil {
		sharedCache = NewLRUCache(web)
	}
	return sharedCache
}

func newWebFetchClient() *http.Client {
	dialer := &net.Dialer{Timeout: 30 * time.Second}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			if err := permission.CheckResolvedFetchHost(host); err != nil {
				return nil, err
			}
			return dialer.DialContext(ctx, network, net.JoinHostPort(host, port))
		},
	}
	return &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}
