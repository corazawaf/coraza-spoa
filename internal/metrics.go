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

	// action: interruption verdict (deny/drop/redirect) or "allow".
	// application: requested SPOE "app" arg, or default_application's name
	// when fallback handles an unknown app. Bounded by applications[].name.
	actionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "coraza_actions_total",
			Help: "Total number of WAF verdicts by action and application",
		},
		[]string{"action", "application"},
	)

	// rule_id: CRS attack ranges only (isAttackRule); ~400 IDs in CRS v4.
	// severity: types.RuleSeverity.String() - 9-value enum.
	ruleTriggersTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "coraza_rule_triggers_total",
			Help: "Total number of CRS attack-rule matches by rule ID and severity",
		},
		[]string{"rule_id", "severity"},
	)

	anomalyScore = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "coraza_anomaly_score",
			Help:    "Distribution of CRS blocking inbound anomaly scores",
			Buckets: []float64{0, 3, 5, 7, 10, 15, 25, 50, 100},
		},
	)
)
