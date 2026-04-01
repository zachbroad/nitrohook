package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	WebhooksReceived = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "nitrohook_webhooks_received_total",
		Help: "Total number of webhooks received",
	}, []string{"source"})

	IngestDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "nitrohook_webhook_ingest_duration_seconds",
		Help:    "Duration of webhook ingest processing",
		Buckets: prometheus.DefBuckets,
	})

	DeliveriesDispatched = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "nitrohook_deliveries_dispatched_total",
		Help: "Total number of delivery dispatches",
	}, []string{"action_type", "status"})

	DispatchDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "nitrohook_dispatch_duration_seconds",
		Help:    "Duration of action dispatch calls",
		Buckets: prometheus.DefBuckets,
	})

	PendingDeliveries = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "nitrohook_pending_deliveries",
		Help: "Number of pending deliveries found by catch-up poller",
	})

	RetryableAttempts = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "nitrohook_retryable_attempts",
		Help: "Number of retryable attempts found by retry poller",
	})
)
