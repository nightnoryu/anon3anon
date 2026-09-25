package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"anon3anon/pkg/domain"
	"anon3anon/pkg/infrastructure/telegram"
)

func TestServerExportsMemoryMetrics(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	newServerHandler(http.NotFoundHandler(), newAppMetrics()).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/metrics", http.NoBody))
	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Header().Get("Content-Type"), "text/plain")
	assert.Contains(t, recorder.Body.String(), "go_memstats_heap_alloc_bytes")
	assert.Contains(t, recorder.Body.String(), "process_resident_memory_bytes")
}

func TestServerExportsMessageMetrics(t *testing.T) {
	t.Parallel()

	metrics := newAppMetrics()
	metrics.observeMessage("anonymous_message", telegram.OutcomeRateLimited, 125*time.Millisecond)

	recorder := httptest.NewRecorder()
	newServerHandler(http.NotFoundHandler(), metrics).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/metrics", http.NoBody))

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `anon3anon_messages_processed_total{event_type="anonymous_message",outcome="rate_limited"} 1`)
	assert.Contains(t, recorder.Body.String(), `anon3anon_message_duration_seconds_count{event_type="anonymous_message"} 1`)
	assert.Contains(t, recorder.Body.String(), `anon3anon_message_duration_seconds_sum{event_type="anonymous_message"} 0.125`)
}

func TestServerExportsRetentionMetrics(t *testing.T) {
	t.Parallel()

	metrics := newAppMetrics()
	metrics.observeRetentionSweep(domain.PurgeStats{Sessions: 2}, errors.New("sweep failed"))
	recorder := httptest.NewRecorder()
	metrics.handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/metrics", http.NoBody))
	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `anon3anon_retention_sweeps_total{result="error"} 1`)
	assert.Contains(t, recorder.Body.String(), `anon3anon_retention_removed_total{table="sessions"} 2`)
	assert.Contains(t, recorder.Body.String(), "anon3anon_retention_last_success_timestamp_seconds 0")

	metrics.observeRetentionSweep(domain.PurgeStats{Relays: 3}, nil)
	recorder = httptest.NewRecorder()
	metrics.handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/metrics", http.NoBody))
	assert.Contains(t, recorder.Body.String(), `anon3anon_retention_sweeps_total{result="success"} 1`)
	assert.Contains(t, recorder.Body.String(), `anon3anon_retention_removed_total{table="relays"} 3`)
	assert.NotContains(t, recorder.Body.String(), "anon3anon_retention_last_success_timestamp_seconds 0")
}
