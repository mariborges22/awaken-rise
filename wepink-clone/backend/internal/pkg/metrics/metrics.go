package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	PaymentsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "awaken_payments_total",
			Help: "Total number of payments processed",
		},
		[]string{"tenant_id", "status"},
	)

	PaymentRetriesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "awaken_payment_retries_total",
			Help: "Total number of payment retries triggered",
		},
		[]string{"tenant_id"},
	)

	OrderProcessingDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "awaken_order_processing_duration_seconds",
			Help:    "Time spent processing an order",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"tenant_id"},
	)
)
