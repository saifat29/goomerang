package main

import (
	"encoding/hex"
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
