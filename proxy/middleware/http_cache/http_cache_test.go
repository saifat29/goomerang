package http_cache

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/saifat29/goomerang/cache"
	"github.com/saifat29/goomerang/config"
	"github.com/stretchr/testify/assert"
)

func TestCacheMiddlewareMiss(t *testing.T) {
	cfg := &config.HTTPCache{TTL: 5 * time.Minute}
	c := cache.NewMemoryLRU(10*1024*1024, 5*time.Minute)
	mw := New(cfg, c)

	upstreamCalled := false
	upstream := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCalled = true
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("hello"))
		assert.NoError(t, err)
	})

	handler := mw(upstream)
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.True(t, upstreamCalled)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "MISS", rec.Header().Get("X-Cache-Status"))
	assert.Equal(t, "hello", rec.Body.String())
}

func TestCacheMiddlewareHit(t *testing.T) {
	cfg := &config.HTTPCache{TTL: 5 * time.Minute}
	c := cache.NewMemoryLRU(10*1024*1024, 5*time.Minute)
	mw := New(cfg, c)

	callCount := 0
	upstream := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("hello"))
		assert.NoError(t, err)
	})

	handler := mw(upstream)

	// First request - cache miss.
	req1 := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)

	assert.Equal(t, 1, callCount)
	assert.Equal(t, "MISS", rec1.Header().Get("X-Cache-Status"))

	// Second request - cache hit.
	req2 := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	assert.Equal(t, 1, callCount, "upstream should not be called again")
	assert.Equal(t, "HIT", rec2.Header().Get("X-Cache-Status"))
	assert.Equal(t, "hello", rec2.Body.String())
}

func TestCacheMiddlewareBypass(t *testing.T) {
	cfg := &config.HTTPCache{TTL: 5 * time.Minute}
	c := cache.NewMemoryLRU(10*1024*1024, 5*time.Minute)
	mw := New(cfg, c)

	upstreamCalled := false
	upstream := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCalled = true
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("response"))
		assert.NoError(t, err)
	})

	handler := mw(upstream)
	req := httptest.NewRequest(http.MethodPost, "/test", http.NoBody)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.True(t, upstreamCalled)
	assert.Equal(t, "BYPASS", rec.Header().Get("X-Cache-Status"))
}

func TestResponseRecorderWrite(t *testing.T) {
	rec := httptest.NewRecorder()
	rr := newResponseRecorder(rec)

	n, err := rr.Write([]byte("hello"))

	assert.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, 5, len(rr.body))

	n, err = rr.Write([]byte(" world"))

	assert.NoError(t, err)
	assert.Equal(t, 6, n)
	assert.Equal(t, 11, len(rr.body))
}

func TestResponseRecorderWriteHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	rr := newResponseRecorder(rec)
	rr.SetInjectHeader("X-Test", "value")

	rr.WriteHeader(http.StatusNotFound)

	assert.Equal(t, http.StatusNotFound, rr.statusCode)
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, "value", rec.Header().Get("X-Test"))
}

func TestResponseRecorderDefaults(t *testing.T) {
	rr := &responseRecorder{statusCode: http.StatusOK}

	assert.Equal(t, http.StatusOK, rr.statusCode)
	assert.Equal(t, 0, len(rr.body))
}

func TestShouldStoreInCache(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		header    http.Header
		reqHeader http.Header
		want      bool
	}{
		{
			name:   "cacheable response",
			status: http.StatusOK,
			header: http.Header{"Content-Type": {"text/plain"}},
			want:   true,
		},
		{
			name:   "non-200 status",
			status: http.StatusNotFound,
			header: http.Header{},
			want:   false,
		},
		{
			name:   "private cache control",
			status: http.StatusOK,
			header: http.Header{"Cache-Control": {"private"}},
			want:   false,
		},
		{
			name:   "no-cache cache control",
			status: http.StatusOK,
			header: http.Header{"Cache-Control": {"no-cache"}},
			want:   false,
		},
		{
			name:   "no-store cache control",
			status: http.StatusOK,
			header: http.Header{"Cache-Control": {"no-store"}},
			want:   false,
		},
		{
			name:   "set-cookie present",
			status: http.StatusOK,
			header: http.Header{"Set-Cookie": {"session=abc"}},
			want:   false,
		},
		{
			name:      "authorization header",
			status:    http.StatusOK,
			header:    http.Header{},
			reqHeader: http.Header{"Authorization": {"Bearer token"}},
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := newResponseRecorder(httptest.NewRecorder())
			rec.statusCode = tt.status
			for k, vs := range tt.header {
				for _, v := range vs {
					rec.Header().Add(k, v)
				}
			}

			req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
			if tt.reqHeader != nil {
				req.Header = tt.reqHeader
			}

			got := saveToCache(req, rec)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseCacheCtrlHeader(t *testing.T) {
	tests := []struct {
		name   string
		header http.Header
		want   []string
	}{
		{
			name:   "single value",
			header: http.Header{"Cache-Control": {"max-age=300"}},
			want:   []string{"max-age=300"},
		},
		{
			name:   "multiple values",
			header: http.Header{"Cache-Control": {"no-cache, no-store"}},
			want:   []string{"no-cache", "no-store"},
		},
		{
			name:   "multiple headers",
			header: http.Header{"Cache-Control": {"max-age=300", "public"}},
			want:   []string{"max-age=300", "public"},
		},
		{
			name:   "empty",
			header: http.Header{},
			want:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseCacheCtrlHeader(tt.header)
			assert.Equal(t, tt.want, got)
		})
	}
}
