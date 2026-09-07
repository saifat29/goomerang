package cache

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cespare/xxhash/v2"
)

// Key is a composite key which will be used to cache upstream responses.
type Key struct {
	Method   string
	URL      string
	Encoding string
	Language string
}

// NewKeyFromRequest extracts relevant fields from the http.Request and creates the CacheKey.
func NewKeyFromRequest(r *http.Request) Key {
	normalisedURL := normaliseURL(r.URL.String())

	return Key{
		Method:   r.Method,
		URL:      normalisedURL,
		Encoding: r.Header.Get("Accept-Encoding"),
		Language: r.Header.Get("Accept-Language"),
	}
}

// String representation of the key. But using `Hash()` key is recommended.
func (k Key) String() string {
	return fmt.Sprintf("%s%s%s%s", k.Method, k.URL, k.Encoding, k.Language)
}

// Hash uses `xxhash` for hashing which generates fixed length string.
// This is recommended over using plain `String()` output as key.
func (k Key) Hash() string {
	hash := xxhash.Sum64String(fmt.Sprintf("%s%s%s%s", k.Method, k.URL, k.Encoding, k.Language))
	return fmt.Sprintf("%x", hash)
}

// normaliseURL lowercases only the hostname of the URL.
// The `scheme` is already normalised by net/http, and other fields must
// preserve their casing.
func normaliseURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return strings.ToLower(rawURL)
	}
	u.Host = strings.ToLower(u.Host)

	return u.String()
}

// Entry contains the response fields to be cached.
type Entry struct {
	Key        Key
	Headers    http.Header
	Body       []byte
	StatusCode int
	TTL        time.Duration
	AccessedAt time.Time
}

// NewEntry accepts the CacheKey and various response fields to create a new entry for caching.
func NewEntry(key Key, code int, header http.Header, body []byte, ttl time.Duration) *Entry {
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
