package internal

import (
	"errors"
	"strconv"

	"github.com/corazawaf/coraza/v3/experimental/plugins/plugintypes"
	"github.com/corazawaf/coraza/v3/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	rulesetInfo = prometheus.NewDesc(
		"coraza_ruleset_info",
		"Ruleset versions observed while loading the active application configuration; always 1.",
		[]string{"application", "ruleset", "version"}, nil,
	)

	handleSPOEDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "coraza_handle_spoe_duration_seconds",
		Help:    "Duration of Coraza SPOE message handling by application, phase, and handler result.",
		Buckets: prometheus.DefBuckets,
	}, []string{"application", "phase", "result"})

	requestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "coraza_requests_total",
		Help: "Total number of HTTP requests received for WAF evaluation.",
	}, []string{"application"})

	transactionsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "coraza_transactions_total",
		Help: "Total number of finished WAF transactions by evaluation outcome, SPOE mode, application, and suspicious classification; outcomes do not imply HAProxy enforcement.",
	}, []string{"application", "mode", "outcome", "suspicious"})

	ruleMatchesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "coraza_rule_matches_total",
		Help: "Total number of matched rules in finished WAF transactions.",
	}, []string{"application", "rule_id", "severity"})

	inboundAnomalyScore = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "coraza_inbound_anomaly_score",
		Help:    "CRS blocking inbound anomaly scores observed at transaction completion, when available.",
		Buckets: []float64{0, 3, 5, 7, 10, 15, 25, 50, 100},
	}, []string{"application"})
)

func (a *Agent) Describe(ch chan<- *prometheus.Desc) {
	ch <- rulesetInfo
}

func (a *Agent) Collect(ch chan<- prometheus.Metric) {
	a.mtx.RLock()
	apps := a.Applications
	a.mtx.RUnlock()
	for name, app := range apps {
		for ruleset := range app.rulesets {
			ch <- prometheus.MustNewConstMetric(rulesetInfo, prometheus.GaugeValue, 1, name, ruleset.name, ruleset.version)
		}
	}
}

// Called by the owner of a transaction after logging and before Close, exactly
// once, including asynchronous responses and expired response correlations.
// Keep this independent of exporting SPOE variables and error-log callbacks.
func recordTransactionMetrics(tx types.Transaction, evaluationErr error, application string, detectOnly, expired bool) {
	outcome := "allow"
	var interruption ErrInterrupted
	switch {
	case expired:
		outcome = "expired"
	case evaluationErr != nil && !errors.As(evaluationErr, &interruption):
		outcome = "error"
	case tx.IsInterrupted():
		outcome = "interrupted"
		switch action := tx.Interruption().Action; action {
		case "deny", "drop", "redirect":
			outcome = action
		}
	}
	mode := "enforce"
	if detectOnly {
		mode = "detect_only"
	}
	suspicious := "false"
	// Every completed transaction contributes to exactly one label set, including
	// transactions without a usable score or threshold.
	defer func() {
		transactionsTotal.WithLabelValues(application, mode, outcome, suspicious).Inc()
	}()
	for _, match := range tx.MatchedRules() {
		ruleMatchesTotal.WithLabelValues(application, strconv.Itoa(match.Rule().ID()), match.Rule().Severity().String()).Inc()
	}
	state, ok := tx.(plugintypes.TransactionState)
	if !ok {
		return
	}
	score, ok := parseTransactionScore(state, "blocking_inbound_anomaly_score")
	if !ok {
		return
	}
	inboundAnomalyScore.WithLabelValues(application).Observe(float64(score))
	if outcome == "allow" && score > 0 {
		if threshold, ok := parseTransactionScore(state, "inbound_anomaly_score_threshold"); ok && score < threshold {
			suspicious = "true"
		}
	}
}

func parseTransactionScore(tx plugintypes.TransactionState, name string) (int64, bool) {
	values := tx.Variables().TX().Get(name)
	if len(values) == 0 {
		return 0, false
	}
	value, err := strconv.ParseInt(values[0], 10, 64)
	return value, err == nil && value >= 0
}
