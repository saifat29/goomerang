package main

import (
	"fmt"
	"net/http"

	"github.com/cespare/xxhash/v2"
)

// CacheKey is a composite key which will be used to cache upstream responses.
type CacheKey struct {
	Method   string
	URL      string
	Encoding string
	Language string
}

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
