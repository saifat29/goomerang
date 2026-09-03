package main

import (
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func newTestResponse(body string) *Response {
	return &Response{
		Headers:    http.Header{"Content-Type": {"text/plain"}},
		Body:       []byte(body),
		StatusCode: http.StatusOK,
		AccessedAt: time.Now(),
		TTL:        time.Hour,
	}
}

func lruOrder(c *MemoryLRU) []string {
	var order []string
	for e := c.rankList.Front(); e != nil; e = e.Next() {
		if res, ok := e.Value.(*Response); ok {
			order = append(order, string(res.Body))
		}
	}
	return order
}

func TestMemoryLRUGetEmptyCache(t *testing.T) {
	t.Run("returns nil for missing key", func(t *testing.T) {
		cache := NewMemoryLRU(10, time.Hour)

		got := cache.Get(CacheKey{Method: "GET", URL: "http://example.com/users"})

		assert.Nil(t, got)
		assert.Empty(t, cache.items, "cache should remain empty")
		assert.Zero(t, cache.rankList.Len(), "rank list should remain empty")
	})

	t.Run("returns nil on zero capacity cache", func(t *testing.T) {
		cache := NewMemoryLRU(0, time.Hour)

		got := cache.Get(CacheKey{Method: "GET", URL: "http://example.com/users"})

		assert.Nil(t, got)
	})
}

func TestMemoryLRUSetThenGet(t *testing.T) {
	cache := NewMemoryLRU(10, time.Hour)
	key := CacheKey{Method: "GET", URL: "http://example.com/users", Encoding: "gzip", Language: "en-US"}
	want := newTestResponse("users")

	cache.Set(key, want)

	got := cache.Get(key)

	assert.Same(t, want, got, "Get should return the exact stored response")
	assert.Equal(t, "users", string(got.Body))
	assert.Equal(t, http.StatusOK, got.StatusCode)
	assert.Equal(t, "text/plain", got.Headers.Get("Content-Type"))
	assert.Equal(t, 1, len(cache.items), "cache should hold exactly one entry")
}

func TestMemoryLRUGetMissingKey(t *testing.T) {
	cache := NewMemoryLRU(10, time.Hour)

	cache.Set(CacheKey{Method: "GET", URL: "http://example.com/users"}, newTestResponse("users"))

	got := cache.Get(CacheKey{Method: "GET", URL: "http://example.com/orders"})

	assert.Nil(t, got, "Get should return nil for a key that was never set")
	assert.Equal(t, 1, len(cache.items), "existing entry should be untouched")
}

func TestMemoryLRUOverwriteSameKey(t *testing.T) {
	cache := NewMemoryLRU(10, time.Hour)
	key := CacheKey{Method: "GET", URL: "http://example.com/users"}

	first := newTestResponse("first")
	second := newTestResponse("second")

	cache.Set(key, first)
	cache.Set(key, second)

	got := cache.Get(key)

	assert.Same(t, second, got, "Get should return the most recently stored response")
	assert.Equal(t, 1, len(cache.items), "overwriting must not duplicate map entries")
	assert.Equal(t, 1, cache.rankList.Len(), "overwriting must not duplicate rank list entries")
}

func TestMemoryLRURecencyOrder(t *testing.T) {
	cache := NewMemoryLRU(10, time.Hour)

	keyA := CacheKey{Method: "GET", URL: "http://example.com/a"}
	keyB := CacheKey{Method: "GET", URL: "http://example.com/b"}
	keyC := CacheKey{Method: "GET", URL: "http://example.com/c"}

	resA := newTestResponse("a")
	resB := newTestResponse("b")
	resC := newTestResponse("c")

	cache.Set(keyA, resA)
	cache.Set(keyB, resB)
	cache.Set(keyC, resC)
	assert.Equal(t, []string{"c", "b", "a"}, lruOrder(cache), "last inserted key should be at the front")

	cache.Get(keyA)
	assert.Equal(t, []string{"a", "c", "b"}, lruOrder(cache), "Get should promote the accessed key to the front")

	resB2 := newTestResponse("b2")
	cache.Set(keyB, resB2)
	assert.Equal(t, []string{"b2", "a", "c"}, lruOrder(cache), "overwriting a key should promote it to the front")
}

func TestMemoryLRUConcurrentAccess(t *testing.T) {
	const (
		goroutines = 8
		iterations = 100
	)

	cache := NewMemoryLRU(goroutines, time.Hour)

	var wg sync.WaitGroup
	for g := range goroutines {
		wg.Add(1)

		go func() {
			defer wg.Done()

			key := CacheKey{Method: "GET", URL: fmt.Sprintf("http://example.com/goroutine-%d", g)}
			neighbor := CacheKey{Method: "GET", URL: fmt.Sprintf("http://example.com/goroutine-%d", (g+1)%goroutines)}
			res := newTestResponse(fmt.Sprintf("goroutine-%d", g))

			for range iterations {
				cache.Set(key, res)

				got := cache.Get(key)
				if !assert.NotNil(t, got, "own key should always be present after Set") {
					return
				}
				assert.Same(t, res, got)

				cache.Get(neighbor)
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, goroutines, len(cache.items), "all goroutine keys should be cached")
}
