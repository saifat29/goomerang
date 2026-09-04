package main

import (
	"fmt"
	"net/http"
	"strings"
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

func TestMemoryLRUGetEvictsExpiredEntry(t *testing.T) {
	t.Run("evicts entry expired by response TTL", func(t *testing.T) {
		cache := NewMemoryLRU(10, time.Hour)
		key := CacheKey{Method: "GET", URL: "http://example.com/users"}

		cache.Set(key, &Response{
			Headers:    http.Header{"Content-Type": {"text/plain"}},
			Body:       []byte("users"),
			StatusCode: http.StatusOK,
			AccessedAt: time.Now().Add(-2 * time.Hour),
			TTL:        time.Hour,
		})

		got := cache.Get(key)

		assert.Nil(t, got, "expired entry should not be returned")
		assert.Empty(t, cache.items, "expired entry should be removed from the map")
		assert.Zero(t, cache.rankList.Len(), "expired entry should be removed from the rank list")
		assert.Nil(t, cache.Get(key), "subsequent Get should still return nil")
	})

	t.Run("evicts entry expired by global cache TTL", func(t *testing.T) {
		cache := NewMemoryLRU(10, 30*time.Minute)
		key := CacheKey{Method: "GET", URL: "http://example.com/users"}

		cache.Set(key, &Response{
			Headers:    http.Header{"Content-Type": {"text/plain"}},
			Body:       []byte("users"),
			StatusCode: http.StatusOK,
			AccessedAt: time.Now().Add(-time.Hour),
			TTL:        24 * time.Hour,
		})

		got := cache.Get(key)

		assert.Nil(t, got, "entry past the global TTL should not be returned")
		assert.Empty(t, cache.items, "entry should be removed from the map")
		assert.Zero(t, cache.rankList.Len(), "entry should be removed from the rank list")
	})
}

func TestMemoryLRUGetNonResponseEntry(t *testing.T) {
	cache := NewMemoryLRU(10, time.Hour)
	key := CacheKey{Method: "GET", URL: "http://example.com/users"}

	cache.Set(key, newTestResponse("users"))
	cache.items[key].Value = "not a response"

	assert.Nil(t, cache.Get(key), "Get should return nil for a corrupted entry")
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
	cache := NewMemoryLRU(1000, time.Hour)

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

func TestMemoryLRUExpired(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name       string
		cacheTTL   time.Duration
		resTTL     time.Duration
		accessedAt time.Time
		want       bool
	}{
		{
			name:       "returns false for fresh response",
			cacheTTL:   time.Hour,
			resTTL:     time.Hour,
			accessedAt: now,
			want:       false,
		},
		{
			name:       "response TTL governs when shorter and has elapsed",
			cacheTTL:   24 * time.Hour,
			resTTL:     time.Hour,
			accessedAt: now.Add(-2 * time.Hour),
			want:       true,
		},
		{
			name:       "response TTL governs when shorter and is fresh",
			cacheTTL:   24 * time.Hour,
			resTTL:     time.Hour,
			accessedAt: now.Add(-30 * time.Minute),
			want:       false,
		},
		{
			name:       "cache TTL governs when shorter and has elapsed",
			cacheTTL:   30 * time.Minute,
			resTTL:     24 * time.Hour,
			accessedAt: now.Add(-time.Hour),
			want:       true,
		},
		{
			name:       "cache TTL governs when shorter and is fresh",
			cacheTTL:   30 * time.Minute,
			resTTL:     24 * time.Hour,
			accessedAt: now.Add(-5 * time.Minute),
			want:       false,
		},
		{
			name:       "expired when both TTLs are equal and elapsed",
			cacheTTL:   time.Hour,
			resTTL:     time.Hour,
			accessedAt: now.Add(-2 * time.Hour),
			want:       true,
		},
		{
			name:       "expired for zero value response",
			cacheTTL:   time.Hour,
			resTTL:     0,
			accessedAt: time.Time{},
			want:       true,
		},
		{
			name:       "expired for negative response TTL",
			cacheTTL:   24 * time.Hour,
			resTTL:     -time.Hour,
			accessedAt: now,
			want:       true,
		},
		{
			name:       "zero cache TTL applies no global cap and response TTL governs",
			cacheTTL:   0,
			resTTL:     time.Hour,
			accessedAt: now.Add(-2 * time.Hour),
			want:       true,
		},
		{
			name:       "zero cache TTL applies no global cap and fresh response stays",
			cacheTTL:   0,
			resTTL:     24 * time.Hour,
			accessedAt: now.Add(-time.Hour),
			want:       false,
		},
		{
			name:       "negative cache TTL applies no global cap",
			cacheTTL:   -time.Hour,
			resTTL:     time.Hour,
			accessedAt: now,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewMemoryLRU(10, tt.cacheTTL)

			got := cache.expired(&Response{
				AccessedAt: tt.accessedAt,
				TTL:        tt.resTTL,
			})

			assert.Equal(t, tt.want, got, "expiry check should match expected")
		})
	}
}

func TestMemoryLRUConcurrentAccess(t *testing.T) {
	const (
		goroutines = 8
		iterations = 100
	)

	cache := NewMemoryLRU(1<<20, time.Hour)

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

func TestMemoryLRUSetEvictsToRespectMaxSize(t *testing.T) {
	t.Run("evicts least recently used entries until the new response fits", func(t *testing.T) {
		cache := NewMemoryLRU(70, time.Hour)

		keyA := CacheKey{Method: "GET", URL: "http://example.com/a"}
		keyB := CacheKey{Method: "GET", URL: "http://example.com/b"}
		keyC := CacheKey{Method: "GET", URL: "http://example.com/c"}
		keyD := CacheKey{Method: "GET", URL: "http://example.com/d"}

		bodyA := strings.Repeat("a", 20)
		bodyB := strings.Repeat("b", 20)
		bodyC := strings.Repeat("c", 20)
		bodyD := strings.Repeat("d", 25)

		resA := NewResponse(keyA, http.StatusOK, http.Header{}, []byte(bodyA), time.Hour)
		resB := NewResponse(keyB, http.StatusOK, http.Header{}, []byte(bodyB), time.Hour)
		resC := NewResponse(keyC, http.StatusOK, http.Header{}, []byte(bodyC), time.Hour)
		resD := NewResponse(keyD, http.StatusOK, http.Header{}, []byte(bodyD), time.Hour)

		cache.Set(keyA, resA)
		cache.Set(keyB, resB)
		cache.Set(keyC, resC)
		cache.Set(keyD, resD)

		assert.Equal(t, []string{bodyD, bodyC, bodyB}, lruOrder(cache), "a should be evicted as the least recently used entry")
		assert.Equal(t, 3, len(cache.items), "only entries that fit should remain")
		assert.Equal(t, 3, cache.rankList.Len(), "rank list should stay in sync with the map")
		assert.Equal(t, 65, cache.usedSizeBytes, "usedSizeBytes should equal the sum of live entry sizes")

		assert.Nil(t, cache.Get(keyA), "evicted entry should not be returned")
		assert.Same(t, resB, cache.Get(keyB))
		assert.Same(t, resC, cache.Get(keyC))
		assert.Same(t, resD, cache.Get(keyD))
	})

	t.Run("keeps recently accessed entries over stale ones", func(t *testing.T) {
		cache := NewMemoryLRU(70, time.Hour)

		keyA := CacheKey{Method: "GET", URL: "http://example.com/a"}
		keyB := CacheKey{Method: "GET", URL: "http://example.com/b"}
		keyC := CacheKey{Method: "GET", URL: "http://example.com/c"}
		keyD := CacheKey{Method: "GET", URL: "http://example.com/d"}

		resA := NewResponse(keyA, http.StatusOK, http.Header{}, make([]byte, 20), time.Hour)
		resB := NewResponse(keyB, http.StatusOK, http.Header{}, make([]byte, 20), time.Hour)
		resC := NewResponse(keyC, http.StatusOK, http.Header{}, make([]byte, 20), time.Hour)
		resD := NewResponse(keyD, http.StatusOK, http.Header{}, make([]byte, 25), time.Hour)

		cache.Set(keyA, resA)
		cache.Set(keyB, resB)
		cache.Set(keyC, resC)

		cache.Get(keyA)

		cache.Set(keyD, resD)

		assert.Nil(t, cache.Get(keyB), "stale entry should be evicted first")
		assert.Same(t, resA, cache.Get(keyA), "recently accessed entry should survive eviction")
		assert.Same(t, resC, cache.Get(keyC))
		assert.Same(t, resD, cache.Get(keyD))
	})
}

func TestMemoryLRUSetEvictsOnExactCapacityBoundary(t *testing.T) {
	cache := NewMemoryLRU(46, time.Hour)

	keyA := CacheKey{Method: "GET", URL: "http://example.com/a"}
	keyB := CacheKey{Method: "GET", URL: "http://example.com/b"}

	resA := NewResponse(keyA, http.StatusOK, http.Header{}, make([]byte, 20), time.Hour)
	resB := NewResponse(keyB, http.StatusOK, http.Header{}, make([]byte, 26), time.Hour)

	cache.Set(keyA, resA)
	cache.Set(keyB, resB)

	assert.Nil(t, cache.Get(keyA), "entry should be evicted when the new response exactly fills capacity")
	assert.Same(t, resB, cache.Get(keyB))
	assert.Equal(t, 1, len(cache.items))
	assert.Equal(t, 26, cache.usedSizeBytes)
}

func TestMemoryLRUSetZeroCapacity(t *testing.T) {
	cache := NewMemoryLRU(0, time.Hour)

	keyA := CacheKey{Method: "GET", URL: "http://example.com/a"}
	keyB := CacheKey{Method: "GET", URL: "http://example.com/b"}

	resA := NewResponse(keyA, http.StatusOK, http.Header{}, make([]byte, 20), time.Hour)
	resB := NewResponse(keyB, http.StatusOK, http.Header{}, make([]byte, 20), time.Hour)

	assert.NotPanics(t, func() {
		cache.Set(keyA, resA)
	}, "Set on a zero capacity cache should not panic")
	assert.Same(t, resA, cache.Get(keyA))

	cache.Set(keyB, resB)

	assert.Nil(t, cache.Get(keyA), "zero capacity cache should only hold the newest entry")
	assert.Same(t, resB, cache.Get(keyB))
	assert.Equal(t, 1, len(cache.items))
	assert.Equal(t, 20, cache.usedSizeBytes)
}

func TestMemoryLRUSetOversizedResponse(t *testing.T) {
	cache := NewMemoryLRU(10, time.Hour)

	keyA := CacheKey{Method: "GET", URL: "http://example.com/a"}
	keyB := CacheKey{Method: "GET", URL: "http://example.com/b"}

	resA := NewResponse(keyA, http.StatusOK, http.Header{}, make([]byte, 20), time.Hour)
	resB := NewResponse(keyB, http.StatusOK, http.Header{}, make([]byte, 30), time.Hour)

	cache.Set(keyA, resA)
	assert.Same(t, resA, cache.Get(keyA), "a single oversized response should still be cached")

	cache.Set(keyB, resB)

	assert.Nil(t, cache.Get(keyA), "all previous entries should be evicted for an oversized response")
	assert.Same(t, resB, cache.Get(keyB))
	assert.Equal(t, 1, len(cache.items))
	assert.Equal(t, 30, cache.usedSizeBytes)
}

func TestMemoryLRUSetZeroSizeResponse(t *testing.T) {
	cache := NewMemoryLRU(10, time.Hour)

	keyA := CacheKey{Method: "GET", URL: "http://example.com/a"}
	keyEmpty := CacheKey{Method: "GET", URL: "http://example.com/empty"}

	resA := NewResponse(keyA, http.StatusOK, http.Header{}, make([]byte, 20), time.Hour)
	resEmpty := NewResponse(keyEmpty, http.StatusOK, http.Header{}, []byte{}, time.Hour)

	cache.Set(keyA, resA)
	cache.Set(keyEmpty, resEmpty)

	assert.Same(t, resEmpty, cache.Get(keyEmpty))
	assert.Nil(t, cache.Get(keyA), "existing entry should be evicted to admit the zero size response")
	assert.Equal(t, 1, len(cache.items))
	assert.Equal(t, 0, cache.usedSizeBytes)
}

func TestMemoryLRUSetOverwriteAccounting(t *testing.T) {
	t.Run("does not double count size when overwriting the same key", func(t *testing.T) {
		cache := NewMemoryLRU(100, time.Hour)
		key := CacheKey{Method: "GET", URL: "http://example.com/users"}

		big := NewResponse(key, http.StatusOK, http.Header{}, make([]byte, 60), time.Hour)
		small := NewResponse(key, http.StatusOK, http.Header{}, make([]byte, 10), time.Hour)

		cache.Set(key, big)
		assert.Equal(t, 60, cache.usedSizeBytes)

		cache.Set(key, small)
		assert.Same(t, small, cache.Get(key))
		assert.Equal(t, 1, len(cache.items), "overwriting must not duplicate map entries")
		assert.Equal(t, 10, cache.usedSizeBytes, "usedSizeBytes should only account for the live entry")

		cache.Set(key, big)
		assert.Same(t, big, cache.Get(key))
		assert.Equal(t, 60, cache.usedSizeBytes)
	})

	t.Run("does not evict entries when the replaced key frees enough space", func(t *testing.T) {
		cache := NewMemoryLRU(50, time.Hour)

		keyA := CacheKey{Method: "GET", URL: "http://example.com/a"}
		keyB := CacheKey{Method: "GET", URL: "http://example.com/b"}

		resA := NewResponse(keyA, http.StatusOK, http.Header{}, make([]byte, 20), time.Hour)
		resB1 := NewResponse(keyB, http.StatusOK, http.Header{}, make([]byte, 20), time.Hour)
		resB2 := NewResponse(keyB, http.StatusOK, http.Header{}, make([]byte, 25), time.Hour)

		cache.Set(keyA, resA)
		cache.Set(keyB, resB1)
		cache.Set(keyB, resB2)

		assert.Same(t, resA, cache.Get(keyA), "entry should survive since the replaced key frees its own space")
		assert.Same(t, resB2, cache.Get(keyB))
		assert.Equal(t, 2, len(cache.items))
		assert.Equal(t, 45, cache.usedSizeBytes)
	})
}

func TestMemoryLRUGetExpiredReclaimsUsedSizeBusedSizeBytes(t *testing.T) {
	cache := NewMemoryLRU(100, time.Hour)

	keyA := CacheKey{Method: "GET", URL: "http://example.com/a"}
	keyB := CacheKey{Method: "GET", URL: "http://example.com/b"}
	keyC := CacheKey{Method: "GET", URL: "http://example.com/c"}

	resA := NewResponse(keyA, http.StatusOK, http.Header{}, make([]byte, 60), time.Hour)
	resB := NewResponse(keyB, http.StatusOK, http.Header{}, make([]byte, 30), time.Hour)
	resC := NewResponse(keyC, http.StatusOK, http.Header{}, make([]byte, 60), time.Hour)

	cache.Set(keyA, resA)
	cache.Set(keyB, resB)

	cache.items[keyA].Value.(*Response).AccessedAt = time.Now().Add(-2 * time.Hour)

	assert.Nil(t, cache.Get(keyA), "expired entry should be evicted")
	assert.Equal(t, 30, cache.usedSizeBytes, "expired eviction should reclaim the entry size")

	cache.Set(keyC, resC)

	assert.Same(t, resB, cache.Get(keyB), "live entry should not be evicted once expired size is reclaimed")
	assert.Same(t, resC, cache.Get(keyC))
	assert.Equal(t, 2, len(cache.items))
	assert.Equal(t, 90, cache.usedSizeBytes)
}

func TestMemoryLRUSweepSkipsCorruptedEntry(t *testing.T) {
	cache := NewMemoryLRU(30, time.Hour)

	keyA := CacheKey{Method: "GET", URL: "http://example.com/a"}
	keyB := CacheKey{Method: "GET", URL: "http://example.com/b"}

	resA := NewResponse(keyA, http.StatusOK, http.Header{}, make([]byte, 20), time.Hour)
	resB := NewResponse(keyB, http.StatusOK, http.Header{}, make([]byte, 20), time.Hour)

	cache.Set(keyA, resA)
	cache.items[keyA].Value = "not a response"

	cache.Set(keyB, resB)

	assert.Same(t, resB, cache.Get(keyB), "Set should still succeed when sweep cannot release space")
	assert.Nil(t, cache.Get(keyA))
	assert.Equal(t, 2, len(cache.items), "corrupted entry should be left untouched by sweep")
	assert.Equal(t, 2, cache.rankList.Len())
	assert.Equal(t, 40, cache.usedSizeBytes)
}

func TestMemoryLRUConcurrentEviction(t *testing.T) {
	const (
		goroutines = 8
		iterations = 200
	)

	cache := NewMemoryLRU(128, time.Hour)

	var wg sync.WaitGroup
	for g := range goroutines {
		wg.Add(1)

		go func() {
			defer wg.Done()

			key := CacheKey{Method: "GET", URL: fmt.Sprintf("http://example.com/evict-%d", g)}

			for range iterations {
				res := NewResponse(key, http.StatusOK, http.Header{}, make([]byte, 20), time.Hour)
				cache.Set(key, res)
				cache.Get(key)
			}
		}()
	}
	wg.Wait()

	cache.mu.Lock()
	defer cache.mu.Unlock()

	assert.Equal(t, cache.rankList.Len(), len(cache.items), "map and rank list should stay in sync")

	totalSize := 0
	for _, item := range cache.items {
		if res, ok := item.Value.(*Response); ok {
			totalSize += res.SizeInBytes()
		}
	}

	assert.Equal(t, totalSize, cache.usedSizeBytes, "usedSizeBytes should equal the sum of live entry sizes")
	assert.Less(t, cache.usedSizeBytes, 128, "usedSizeBytes should stay under max size after every Set")
}
