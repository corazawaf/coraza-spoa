package internal

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	handleSPOEDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "coraza_handle_spoe_duration_seconds",
			Help:    "Duration of Coraza SPOE handling",
			Buckets: prometheus.DefBuckets,
		},
	)
	responseUncorrelatedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "coraza_response_uncorrelated_total",
			Help: "Number of coraza-res messages that could not be matched to a transaction, by reason",
		},
		[]string{"reason"},
	)
)
