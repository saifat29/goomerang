package cache

import (
	"fmt"
	"net/http"
	"time"

	"github.com/cespare/xxhash/v2"
)

// CacheKey is a composite key which will be used to cache upstream responses.
type CacheKey struct {
	Method   string
	URL      string
	Encoding string
	Language string
}

// NewCacheKey extracts relevant fields from the http.Request and creates the CacheKey.
func NewCacheKey(r *http.Request) CacheKey {
	return CacheKey{
		Method:   r.Method,
		URL:      r.URL.String(),
		Encoding: r.Header.Get("Accept-Encoding"),
		Language: r.Header.Get("Accept-Language"),
	}
}

// String representation of the key. But using `Hash()` key is recommended.
func (k CacheKey) String() string {
	return fmt.Sprintf("%s%s%s%s", k.Method, k.URL, k.Encoding, k.Language)
}

// Hash uses `xxhash` for hashing which generates fixed length string.
// This is recommended over using plain `String()` output as key.
func (k CacheKey) Hash() string {
	hash := xxhash.Sum64String(fmt.Sprintf("%s%s%s%s", k.Method, k.URL, k.Encoding, k.Language))
	return fmt.Sprintf("%x", hash)
}

// Entry contains the response fields to be cached.
type Entry struct {
	Key        CacheKey
	Headers    http.Header
	Body       []byte
	StatusCode int
	TTL        time.Duration
	AccessedAt time.Time
}

// NewEntry accepts the CacheKey and various response fields to create a new entry for caching.
func NewEntry(key CacheKey, code int, header http.Header, body []byte, ttl time.Duration) *Entry {
	bodyCopy := make([]byte, len(body))
	copy(bodyCopy, body)

	et := &Entry{
		Key:        key,
		Headers:    header.Clone(),
		Body:       bodyCopy,
		StatusCode: code,
		TTL:        ttl,
		AccessedAt: time.Now().UTC(),
	}

	return et
}

// SizeInBytes returns the size of cached response in bytes.
// Only `Headers` and `Body` is considered for size calculation,
// because the other fields (and struct padding) would comparatively
// take insignificant space. This is a good enough solution.
func (et *Entry) SizeInBytes() int {
	size := 0

	for key, values := range et.Headers {
		size += len(key)
		for _, value := range values {
			size += len(value)
		}
	}

	size += cap(et.Body)

	return size
}
