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
)
