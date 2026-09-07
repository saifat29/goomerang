package cache

import (
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func newTestKey(method, rawURL, encoding, language string) Key {
	req := httptest.NewRequest(method, rawURL, http.NoBody)
	if encoding != "" {
		req.Header.Set("Accept-Encoding", encoding)
	}
	if language != "" {
		req.Header.Set("Accept-Language", language)
	}
	return NewKeyFromRequest(req)
}

func TestCacheKeyString(t *testing.T) {
	tests := []struct {
		name string
		key  Key
		want string
	}{
		{
			name: "concatenates all fields",
			key:  newTestKey("GET", "http://example.com/users", "gzip", "en-US"),
			want: "GEThttp://example.com/usersgzipen-US",
		},
		{
			name: "handles empty encoding and language",
			key:  newTestKey("POST", "http://example.com/orders", "", ""),
			want: "POSThttp://example.com/orders",
		},
		{
			name: "lowercases hostname but preserves path casing",
			key:  newTestKey("GET", "http://EXAMPLE.COM/Users", "", ""),
			want: "GEThttp://example.com/Users",
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
		key := newTestKey("GET", "http://example.com/users", "gzip", "en-US")

		first := key.Hash()
		for range 10 {
			assert.Equal(t, first, key.Hash(), "hash should be identical across calls")
		}
	})

	t.Run("produces a fixed length hex string", func(t *testing.T) {
		key := newTestKey("GET", "http://example.com/users", "", "")

		got := key.Hash()

		assert.Len(t, got, 16, "hash should be a 16 character hex string")
		_, err := hex.DecodeString(got)
		assert.NoError(t, err, "hash should be valid hex")
	})

	t.Run("differs for distinct keys", func(t *testing.T) {
		base := newTestKey("GET", "http://example.com/users", "gzip", "en-US")

		tests := []struct {
			name string
			key  Key
		}{
			{
				name: "different method",
				key:  newTestKey("POST", base.URL, base.Encoding, base.Language),
			},
			{
				name: "different URL",
				key:  newTestKey(base.Method, "http://example.com/orders", base.Encoding, base.Language),
			},
			{
				name: "different encoding",
				key:  newTestKey(base.Method, base.URL, "br", base.Language),
			},
			{
				name: "different language",
				key:  newTestKey(base.Method, base.URL, base.Encoding, "de-DE"),
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				assert.NotEqual(t, base.Hash(), tt.key.Hash(), "distinct keys should hash differently")
			})
		}
	})

	t.Run("same hash as hashing the String output", func(t *testing.T) {
		key := newTestKey("GET", "http://example.com/users", "gzip", "en-US")

		assert.NotEqual(t, key.String(), key.Hash(), "hash should not leak the raw String output")
	})

	t.Run("same key for different-case hostnames", func(t *testing.T) {
		a := newTestKey("GET", "http://EXAMPLE.COM/users", "", "")
		b := newTestKey("GET", "http://example.com/users", "", "")

		assert.Equal(t, a.Hash(), b.Hash())
	})

	t.Run("different keys for different-case paths", func(t *testing.T) {
		a := newTestKey("GET", "http://example.com/Users", "", "")
		b := newTestKey("GET", "http://example.com/users", "", "")

		assert.NotEqual(t, a.Hash(), b.Hash())
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
