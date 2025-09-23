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

	// Counter for the number of SPOE requests processed
	handleSPOECount = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "coraza_handle_spoe_count",
			Help: "Total number of SPOE requests handled",
		},
	)

	// Counter of responses by severity
	handleResponsesBySeverity = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "coraza_handle_responses_severity",
			Help: "Number of responses by severity",
		},
		[]string{"severity"},
	)

	// Counter of responses by rule
	handleResponsesByRule = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "coraza_handle_responses_rules",
			Help: "Number of responses by rule ID",
		},
		[]string{"rule_id"},
	)

	// Gauge for OWASP CRS version
	handleVersion = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "coraza_handle_version",
			Help: "OWASP CRS version information",
		},
		[]string{"version"},
	)
)
