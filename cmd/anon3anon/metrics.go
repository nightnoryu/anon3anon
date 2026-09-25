package main

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"anon3anon/pkg/infrastructure/telegram"
)

type appMetrics struct {
	registry          *prometheus.Registry
	messagesProcessed *prometheus.CounterVec
	messageDuration   *prometheus.HistogramVec
}

func newAppMetrics() *appMetrics {
	registry := prometheus.NewRegistry()
	m := &appMetrics{
		registry: registry,
		messagesProcessed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "anon3anon_messages_processed_total",
			Help: "Number of Telegram messages processed by event type and outcome.",
		}, []string{"event_type", "outcome"}),
		messageDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "anon3anon_message_duration_seconds",
			Help:    "Time spent processing a Telegram message by event type.",
			Buckets: prometheus.DefBuckets,
		}, []string{"event_type"}),
	}
	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		m.messagesProcessed,
		m.messageDuration,
	)
	return m
}

func (m *appMetrics) observeMessage(eventType string, outcome telegram.Outcome, duration time.Duration) {
	m.messagesProcessed.WithLabelValues(eventType, string(outcome)).Inc()
	m.messageDuration.WithLabelValues(eventType).Observe(duration.Seconds())
}

func (m *appMetrics) handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}
