package logger

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResponseRecorderWrite(t *testing.T) {
	rec := httptest.NewRecorder()
	rr := &responseRecorder{ResponseWriter: rec, statusCode: http.StatusOK}

	n, err := rr.Write([]byte("hello"))

	assert.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, 5, rr.size)

	n, err = rr.Write([]byte(" world"))

	assert.NoError(t, err)
	assert.Equal(t, 6, n)
	assert.Equal(t, 11, rr.size)
}

func TestResponseRecorderWriteHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	rr := &responseRecorder{ResponseWriter: rec, statusCode: http.StatusOK}

	rr.WriteHeader(http.StatusNotFound)

	assert.Equal(t, http.StatusNotFound, rr.statusCode)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestResponseRecorderDefaults(t *testing.T) {
	rr := &responseRecorder{statusCode: http.StatusOK}

	// No WriteHeader called, default should be 200.
	assert.Equal(t, http.StatusOK, rr.statusCode)
	assert.Equal(t, 0, rr.size)
}
