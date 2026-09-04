package cache

import (
	"encoding/hex"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCacheKeyString(t *testing.T) {
	tests := []struct {
		name string
		key  CacheKey
		want string
	}{
		{
			name: "concatenates all fields",
			key: CacheKey{
				Method:   "GET",
				URL:      "http://example.com/users",
				Encoding: "gzip",
				Language: "en-US",
			},
			want: "GEThttp://example.com/usersgzipen-US",
		},
		{
			name: "returns empty string for zero value key",
			key:  CacheKey{},
			want: "",
		},
		{
			name: "handles empty URL encoding and language",
			key: CacheKey{
				Method: "POST",
				URL:    "http://example.com/orders",
			},
			want: "POSThttp://example.com/orders",
		},
		{
			name: "handles only encoding set",
			key: CacheKey{
				Encoding: "br",
			},
			want: "br",
		},
		{
			name: "handles only language set",
			key: CacheKey{
				Language: "de-DE",
			},
			want: "de-DE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.key.String())
		})
	}
}

func TestCacheKeyHash(t *testing.T) {
	t.Run("is deterministic for the same key", func(t *testing.T) {
		key := CacheKey{Method: "GET", URL: "http://example.com/users", Encoding: "gzip", Language: "en-US"}

		first := key.Hash()
		for range 10 {
			assert.Equal(t, first, key.Hash(), "hash should be identical across calls")
		}
	})

	t.Run("produces a fixed length hex string", func(t *testing.T) {
		key := CacheKey{Method: "GET", URL: "http://example.com/users"}

		got := key.Hash()

		assert.Len(t, got, 16, "hash should be a 16 character hex string")
		_, err := hex.DecodeString(got)
		assert.NoError(t, err, "hash should be valid hex")
	})

	t.Run("differs for distinct keys", func(t *testing.T) {
		base := CacheKey{Method: "GET", URL: "http://example.com/users", Encoding: "gzip", Language: "en-US"}

		tests := []struct {
			name string
			key  CacheKey
		}{
			{
				name: "different method",
				key:  CacheKey{Method: "POST", URL: base.URL, Encoding: base.Encoding, Language: base.Language},
			},
			{
				name: "different URL",
				key:  CacheKey{Method: base.Method, URL: "http://example.com/orders", Encoding: base.Encoding, Language: base.Language},
			},
			{
				name: "different encoding",
				key:  CacheKey{Method: base.Method, URL: base.URL, Encoding: "br", Language: base.Language},
			},
			{
				name: "different language",
				key:  CacheKey{Method: base.Method, URL: base.URL, Encoding: base.Encoding, Language: "de-DE"},
			},
			{
				name: "empty key",
				key:  CacheKey{},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				assert.NotEqual(t, base.Hash(), tt.key.Hash(), "distinct keys should hash differently")
			})
		}
	})

	t.Run("same hash as hashing the String output", func(t *testing.T) {
		key := CacheKey{Method: "GET", URL: "http://example.com/users", Encoding: "gzip", Language: "en-US"}

		assert.NotEqual(t, key.String(), key.Hash(), "hash should not leak the raw String output")
	})
}

func TestEntrySizeInBytes(t *testing.T) {
	tests := []struct {
		name  string
		entry Entry
		want  int
	}{
		{
			name:  "returns zero for empty entry",
			entry: Entry{},
			want:  0,
		},
		{
			name:  "returns zero for empty header map",
			entry: Entry{Headers: http.Header{}},
			want:  0,
		},
		{
			name:  "counts body capacity even when body is empty",
			entry: Entry{Body: make([]byte, 0, 100)},
			want:  100,
		},
		{
			name:  "counts body capacity not length",
			entry: Entry{Body: make([]byte, 2, 10)},
			want:  10,
		},
		{
			name: "counts header keys and values",
			entry: Entry{
				Headers: http.Header{"X-Foo": {"bar"}},
			},
			want: 8,
		},
		{
			name: "counts every value of a multi value header",
			entry: Entry{
				Headers: http.Header{"X-Foo": {"a", "bb"}},
			},
			want: 8,
		},
		{
			name: "counts all headers",
			entry: Entry{
				Headers: http.Header{"X-Foo": {"bar"}, "X-Baz": {"qux"}},
			},
			want: 16,
		},
		{
			name: "counts headers and body together",
			entry: Entry{
				Headers: http.Header{"Content-Type": {"text/plain"}},
				Body:    make([]byte, 0, 5),
			},
			want: 27,
		},
		{
			name: "counts multibyte values in bytes",
			entry: Entry{
				Headers: http.Header{"X-Lang": {"héllo"}},
			},
			want: 12,
		},
		{
			name: "counts empty key and value as zero",
			entry: Entry{
				Headers: http.Header{"": {""}},
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.entry.SizeInBytes(), "size in bytes should match expected")
		})
	}
}
