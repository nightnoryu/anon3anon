package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerExportsMemoryMetrics(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	newServerHandler(http.NotFoundHandler()).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/metrics", http.NoBody))
	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Header().Get("Content-Type"), "text/plain")
	assert.Contains(t, recorder.Body.String(), "go_memstats_heap_alloc_bytes")
	assert.Contains(t, recorder.Body.String(), "process_resident_memory_bytes")
}
