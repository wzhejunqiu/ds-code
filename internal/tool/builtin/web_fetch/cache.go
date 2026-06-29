package web_fetch

import (
	"bytes"
	"compress/gzip"
	"container/list"
	"io"
	"sync"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/config"
)

const cacheEntryOverheadBytes = 128

// PageBody holds a fetched HTTP response body and metadata.
type PageBody struct {
	Body        []byte
	ContentType string
	StatusCode  int
}

// LRUCache stores gzip-compressed page bodies in memory with LRU eviction.
type LRUCache struct {
	mu         sync.Mutex
	maxBytes   int
	ttl        time.Duration
	totalBytes int
	items      map[string]*list.Element
	order      *list.List
	now        func() time.Time
}

type cacheEntry struct {
	key            string
	compressedBody []byte
	bodyBytes      int
	contentType    string
	statusCode     int
	expiresAt      time.Time
	sizeBytes      int
}

// NewLRUCache creates a cache from web config.
func NewLRUCache(web config.WebConfig) *LRUCache {
	return &LRUCache{
		maxBytes: web.ResolveFetchCacheMaxBytes(),
		ttl:      web.ResolveFetchCacheTTL(),
		items:    make(map[string]*list.Element),
		order:    list.New(),
		now:      time.Now,
	}
}

// Get returns a cached page body or nil if missing/expired.
func (c *LRUCache) Get(key string) *PageBody {
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.items[key]
	if !ok {
		return nil
	}
	ent := el.Value.(*cacheEntry)
	if c.now().After(ent.expiresAt) {
		c.removeElement(el)
		return nil
	}
	body, err := gunzipBytes(ent.compressedBody)
	if err != nil || len(body) != ent.bodyBytes {
		c.removeElement(el)
		return nil
	}
	c.order.MoveToFront(el)
	return &PageBody{
		Body:        body,
		ContentType: ent.contentType,
		StatusCode:  ent.statusCode,
	}
}

// Put stores a page body when allowed.
func (c *LRUCache) Put(key string, page PageBody) {
	if len(page.Body) == 0 || len(page.Body) > maxFetchBodyBytes {
		return
	}
	compressed, err := gzipBytes(page.Body)
	if err != nil {
		return
	}
	ent := &cacheEntry{
		key:            key,
		compressedBody: compressed,
		bodyBytes:      len(page.Body),
		contentType:    page.ContentType,
		statusCode:     page.StatusCode,
		expiresAt:      c.now().Add(c.ttl),
		sizeBytes:      len(compressed) + cacheEntryOverheadBytes,
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		c.totalBytes -= el.Value.(*cacheEntry).sizeBytes
		c.order.Remove(el)
		delete(c.items, key)
	}
	if ent.sizeBytes > c.maxBytes {
		return
	}
	el := c.order.PushFront(ent)
	c.items[key] = el
	c.totalBytes += ent.sizeBytes
	for c.totalBytes > c.maxBytes && c.order.Len() > 0 {
		back := c.order.Back()
		c.removeElement(back)
	}
}

func (c *LRUCache) removeElement(el *list.Element) {
	ent := el.Value.(*cacheEntry)
	c.totalBytes -= ent.sizeBytes
	delete(c.items, ent.key)
	c.order.Remove(el)
}

func gzipBytes(raw []byte) ([]byte, error) {
	var buf bytes.Buffer
	zw, err := gzip.NewWriterLevel(&buf, gzip.BestSpeed)
	if err != nil {
		return nil, err
	}
	if _, err := zw.Write(raw); err != nil {
		_ = zw.Close()
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func gunzipBytes(compressed []byte) ([]byte, error) {
	zr, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	return io.ReadAll(io.LimitReader(zr, maxFetchBodyBytes+1))
}
