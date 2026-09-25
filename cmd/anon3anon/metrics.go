package main

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"anon3anon/pkg/domain"
	"anon3anon/pkg/infrastructure/telegram"
)

type appMetrics struct {
	registry          *prometheus.Registry
	messagesProcessed *prometheus.CounterVec
	messageDuration   *prometheus.HistogramVec
	retentionSweeps   *prometheus.CounterVec
	retentionRemoved  *prometheus.CounterVec
	lastSweepSuccess  prometheus.Gauge
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
		retentionSweeps: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "anon3anon_retention_sweeps_total",
			Help: "Number of retention sweeps by result.",
		}, []string{"result"}),
		retentionRemoved: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "anon3anon_retention_removed_total",
			Help: "Number of rows removed by retention sweeps, including partial sweeps.",
		}, []string{"table"}),
		lastSweepSuccess: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "anon3anon_retention_last_success_timestamp_seconds",
			Help: "Unix timestamp of the last successful retention sweep, or zero if none has completed.",
		}),
	}
	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		m.messagesProcessed,
		m.messageDuration,
		m.retentionSweeps,
		m.retentionRemoved,
		m.lastSweepSuccess,
	)
	return m
}

func (m *appMetrics) observeMessage(eventType string, outcome telegram.Outcome, duration time.Duration) {
	m.messagesProcessed.WithLabelValues(eventType, string(outcome)).Inc()
	m.messageDuration.WithLabelValues(eventType).Observe(duration.Seconds())
}

func (m *appMetrics) observeRetentionSweep(stats domain.PurgeStats, err error) {
	m.retentionRemoved.WithLabelValues("sessions").Add(float64(stats.Sessions))
	m.retentionRemoved.WithLabelValues("relays").Add(float64(stats.Relays))
	m.retentionRemoved.WithLabelValues("blocks").Add(float64(stats.Blocks))
	m.retentionRemoved.WithLabelValues("message_rates").Add(float64(stats.MessageRates))
	if err != nil {
		m.retentionSweeps.WithLabelValues("error").Inc()
		return
	}
	m.retentionSweeps.WithLabelValues("success").Inc()
	m.lastSweepSuccess.SetToCurrentTime()
}

func (m *appMetrics) handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}
