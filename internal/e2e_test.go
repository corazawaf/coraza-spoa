//go:build e2e

package internal

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/corazawaf/coraza/v3/http/e2e"
	"github.com/dropmorepackets/haproxy-go/pkg/testutil"
	"github.com/mccutchen/go-httpbin/v2/httpbin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
)

func TestE2E(t *testing.T) {
	t.Run("coraza e2e suite", func(t *testing.T) {
		config, bin, _ := runCoraza(t, e2e.Directives)
		err := e2e.Run(e2e.Config{
			NulledBody:        false,
			ProxiedEntrypoint: "http://127.0.0.1:" + config.FrontendPort,
			HttpbinEntrypoint: bin,
		})
		if err != nil {
			t.Fatalf("e2e tests failed: %v", err)
		}
	})
	t.Run("high request rate", func(t *testing.T) {
		config, _, _ := runCoraza(t, e2e.Directives)

		if os.Getenv("CI") != "" {
			t.Skip("CI is too slow for this test.")
		}

		before := gatherRequestMetrics(t, "allow", "enforce")
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := 0; i < 100; i++ {
					req, _ := http.NewRequest("GET", "http://127.0.0.1:"+config.FrontendPort+"/get", http.NoBody)
					req.Header.Set("coraza-e2e", "ok")
					resp, _ := http.DefaultClient.Do(req)
					if resp.StatusCode != http.StatusOK {
						t.Error(resp.Status)
					}
				}
			}()
		}

		wg.Wait()
		after := gatherRequestMetrics(t, "allow", "enforce")
		if after.requests-before.requests != 1000 || after.transactions-before.transactions != 1000 {
			t.Errorf("expected 1000 requests and completions, got %v and %v", after.requests-before.requests, after.transactions-before.transactions)
		}
	})

	const defaultCorazaConfig = `
Include @coraza.conf-recommended
Include @crs-setup.conf.example
Include @owasp_crs/*.conf
SecRule REQUEST_HEADERS:coraza-e2e "@streq ok" "id:1234567,phase:1,pass,nolog,ver:'local/1.0.0'"
SecRuleEngine On
`
	t.Run("detect-only", func(t *testing.T) {
		config, _, _ := runCorazaDetectOnly(t, defaultCorazaConfig)

		t.Run("clean request passes", func(t *testing.T) {
			checkMetrics := checkRequestMetrics(t, "allow", "enforce", false)
			// We have to access via localhost to prevent 920350 matching.
			req, _ := http.NewRequest("GET", "http://localhost:"+config.FrontendPort+"/", http.NoBody)
			req.Header.Set("coraza-e2e", "ok")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("expected status code to be \"%d\", but got \"%d\"", http.StatusOK, resp.StatusCode)
			}
			checkMetrics(resp)
		})

		t.Run("request phase still blocks", func(t *testing.T) {
			checkMetrics := checkRequestMetrics(t, "deny", "enforce", false)
			req, _ := http.NewRequest("GET", "http://127.0.0.1:"+config.FrontendPort+"/anything?arg=<script>alert(0)</script>", http.NoBody)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusForbidden {
				t.Errorf("expected status code to be \"%d\", but got \"%d\"", http.StatusForbidden, resp.StatusCode)
			}
			checkMetrics(resp)
		})
	})

	t.Run("request detect-only forwards and correlates response", func(t *testing.T) {
		// Full detect-only: HAProxy does not enforce the verdict, so a request
		// the WAF would deny is forwarded to the origin and its response
		// returned. The interrupted request transaction must still be cached so
		// the response can be correlated; otherwise HandleResponse fails with
		// "transaction not found" and the request is denied with a 504.
		config, _, _ := runCorazaRequestDetectOnly(t, defaultCorazaConfig)
		checkMetrics := checkRequestMetrics(t, "deny", "detect_only", false)

		req, _ := http.NewRequest("GET", "http://127.0.0.1:"+config.FrontendPort+"/anything?arg=<script>alert(0)</script>", http.NoBody)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		// Forwarded to the origin (httpbin /anything echoes with 200), proving
		// the request was not blocked and the response was correlated/logged
		// rather than lost (which would yield a 504).
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status code to be \"%d\", but got \"%d\"", http.StatusOK, resp.StatusCode)
		}

		// The WAF still detected the malicious request (request-phase metrics
		// are exported even when the verdict is not enforced).
		if ruleIDs := resp.Header.Get("X-Rule-IDs"); ruleIDs == "" {
			t.Errorf("expected rule_ids to be not empty (request should still be detected)")
		}
		checkMetrics(resp)
	})

	t.Run("ruleset versions on replacement", func(t *testing.T) {
		rulesFile := filepath.Join(t.TempDir(), "custom.conf")
		if err := os.WriteFile(rulesFile, []byte(`SecRule REQUEST_URI "@streq /never-requested" "id:1234567,phase:1,pass,ver:'custom/1.0'"`), 0600); err != nil {
			t.Fatal(err)
		}
		directives := fmt.Sprintf("Include %s\n", rulesFile)
		a, _, _ := setupCorazaAgent(t, directives)
		assertRulesets := func(expected map[string]bool) {
			t.Helper()
			families, err := prometheus.DefaultGatherer.Gather()
			if err != nil {
				t.Fatal(err)
			}
			found := make(map[string]bool)
			for _, family := range families {
				if family.GetName() != "coraza_ruleset_info" {
					continue
				}
				for _, metric := range family.Metric {
					labels := make(map[string]string)
					for _, label := range metric.Label {
						labels[label.GetName()] = label.GetValue()
					}
					found[labels["application"]+":"+labels["ruleset"]+":"+labels["version"]] = true
					if metric.GetGauge().GetValue() != 1 {
						t.Error("ruleset info must be 1")
					}
				}
			}
			if !reflect.DeepEqual(found, expected) {
				t.Fatalf("rulesets: got %v, want %v", found, expected)
			}
		}
		// Versions are visible without executing a single rule.
		assertRulesets(map[string]bool{"default:custom:1.0": true})
		if err := os.WriteFile(rulesFile, []byte(`SecRule REQUEST_URI "@streq /never-requested" "id:1234567,phase:1,pass,ver:'custom/2.0'"
SecRule REQUEST_URI "@streq /also-never-requested" "id:1234568,phase:1,pass,ver:'unqualified-version'"`), 0600); err != nil {
			t.Fatal(err)
		}
		replacement, err := (AppConfig{Name: "replacement", Directives: directives, Logger: zerolog.Nop()}).NewApplication()
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(replacement.cache.stop)
		// Preparing a replacement must not change the active metric.
		assertRulesets(map[string]bool{"default:custom:1.0": true})
		a.ReplaceApplications(map[string]*Application{"replacement": replacement}, replacement)
		assertRulesets(map[string]bool{"replacement:custom:2.0": true, "replacement::unqualified-version": true})
		// Discover a new version before failing. It must never reach the active
		// metric, even though the rule observer ran before the parser error.
		if err := os.WriteFile(rulesFile, []byte(`SecRule REQUEST_URI "@streq /never-requested" "id:1234567,phase:1,pass,ver:'custom/3.0'"
InvalidDirective On`), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := (AppConfig{Directives: directives, Logger: zerolog.Nop()}).NewApplication(); err == nil {
			t.Fatal("expected invalid configuration")
		}
		if a.Applications["replacement"] != replacement {
			t.Fatal("failed load replaced the active application")
		}
		assertRulesets(map[string]bool{"replacement:custom:2.0": true, "replacement::unqualified-version": true})
		a.ReplaceApplications(nil, nil)
		assertRulesets(map[string]bool{})
	})

	t.Run("default config", func(t *testing.T) {
		config, _, _ := runCoraza(t, defaultCorazaConfig)

		t.Run("metrics for clean", func(t *testing.T) {
			checkMetrics := checkRequestMetrics(t, "allow", "enforce", false)
			// We have to access via localhost to prevent 920350 matching.
			req, _ := http.NewRequest("GET", "http://localhost:"+config.FrontendPort+"/", http.NoBody)
			req.Header.Set("coraza-e2e", "ok")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("expected status code to be \"%d\", but got \"%d\"", http.StatusOK, resp.StatusCode)
			}

			if anomalyScore := resp.Header.Get("X-Anomaly-Score"); anomalyScore != "0" {
				t.Errorf("expected X-Anomaly-Score to be %q, got %q", "0", anomalyScore)
			}

			if ruleIDs := resp.Header.Get("X-Rule-IDs"); ruleIDs != "" {
				t.Errorf("expected rule_ids to be empty")
			}
			checkMetrics(resp)
		})

		t.Run("metrics for suspicious", func(t *testing.T) {
			checkMetrics := checkRequestMetrics(t, "allow", "enforce", true)
			// The numeric Host triggers CRS 920350 with a score below the deny threshold.
			req, _ := http.NewRequest("GET", "http://127.0.0.1:"+config.FrontendPort+"/", http.NoBody)
			req.Header.Set("coraza-e2e", "ok")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("expected allowed response, got %d", resp.StatusCode)
			}
			if resp.Header.Get("X-Anomaly-Score") != "3" {
				t.Fatalf("expected score 3, got %q", resp.Header.Get("X-Anomaly-Score"))
			}
			checkMetrics(resp)
		})

		t.Run("metrics for malicious", func(t *testing.T) {
			checkMetrics := checkRequestMetrics(t, "deny", "enforce", false)
			req, _ := http.NewRequest("GET", "http://127.0.0.1:"+config.FrontendPort+"/anything?arg=<script>alert(0)</script>", http.NoBody)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusForbidden {
				t.Errorf("expected status code to be \"%d\", but got \"%d\"", http.StatusForbidden, resp.StatusCode)
			}

			if anomalyScore := resp.Header.Get("X-Anomaly-Score"); anomalyScore == "0" {
				t.Errorf("expected X-Anomaly-Score to not be %q, got %q", "0", anomalyScore)
			}

			if ruleIDs := resp.Header.Get("X-Rule-IDs"); ruleIDs == "" {
				t.Errorf("expected rule_ids to be not empty")
			}
			checkMetrics(resp)
		})
	})

}

func setupCorazaAgent(tb testing.TB, directives string) (*Agent, string, string) {
	s := httptest.NewServer(httpbin.New())
	tb.Cleanup(s.Close)

	logger := zerolog.New(os.Stderr).With().Timestamp().Logger()

	appCfg := AppConfig{
		Name:           "default",
		Directives:     directives,
		ResponseCheck:  true,
		Logger:         logger,
		TransactionTTL: 10 * time.Second,
	}

	application, err := appCfg.NewApplication()
	if err != nil {
		tb.Fatal(err)
	}

	a := &Agent{
		Context: context.Background(),
		Logger:  logger,
	}
	a.ReplaceApplications(map[string]*Application{"default": application}, application)

	tb.Cleanup(func() { a.DrainDetectOnly(); application.cache.stop() })
	return a, s.URL, s.Listener.Addr().String()
}

func runCoraza(tb testing.TB, directives string) (testutil.HAProxyConfig, string, string) {
	a, binURL, backendAddr := setupCorazaAgent(tb, directives)

	// create the listener synchronously to prevent a race
	l := testutil.TCPListener(tb)
	// ignore errors as the listener will be closed by t.Cleanup
	go func() { _ = a.Serve(l) }()

	cfg := testutil.HAProxyConfig{
		EngineAddr:   l.Addr().String(),
		FrontendPort: fmt.Sprintf("%d", testutil.TCPPort(tb)),
		CustomFrontendConfig: `
    # Currently haproxy cannot use variables to set the code or deny_status, so this needs to be manually configured here
    http-request redirect code 302 location %[var(txn.e2e.data)] if { var(txn.e2e.action) -m str redirect }
    http-response redirect code 302 location %[var(txn.e2e.data)] if { var(txn.e2e.action) -m str redirect }

    acl is_deny var(txn.e2e.action) -m str deny
    acl status_424 var(txn.e2e.status) -m int 424

    http-after-response set-header X-Anomaly-Score "%[var(txn.e2e.anomaly_score)]"
    http-after-response set-header X-Rules-Hit "%[var(txn.e2e.rules_hit)]"
    http-after-response set-header X-Rule-IDs "%[var(txn.e2e.rule_ids)]"

    # Special check for e2e tests as they validate the config.
    http-request deny deny_status 424 hdr waf-block "request" if is_deny status_424
    http-response deny deny_status 424 hdr waf-block "response" if is_deny status_424

    http-request deny deny_status 403 hdr waf-block "request" if is_deny
    http-response deny deny_status 403 hdr waf-block "response" if is_deny

    http-request silent-drop if { var(txn.e2e.action) -m str drop }
    http-response silent-drop if { var(txn.e2e.action) -m str drop }

    # Deny in case of an error, when processing with the Coraza SPOA
    http-request deny deny_status 504 if { var(txn.e2e.error) -m int gt 0 }
    http-response deny deny_status 504 if { var(txn.e2e.error) -m int gt 0 }
`,
		EngineConfig: `
[e2e]
spoe-agent e2e
    messages    coraza-req     coraza-res
    option      var-prefix      e2e
    option      set-on-error    error
    timeout     hello           2s
    timeout     idle            2m
    timeout     processing      500ms
    use-backend e2e-spoa
    log         global

spoe-message coraza-req
    args app=str(default) src-ip=src src-port=src_port dst-ip=dst dst-port=dst_port method=method path=path query=query version=req.ver headers=req.hdrs body=req.body exportRuleIDs=bool(true)
    event on-frontend-http-request

spoe-message coraza-res
    args app=str(default) id=var(txn.e2e.id) version=res.ver status=status headers=res.hdrs body=res.body exportRuleIDs=bool(true)
    event on-http-response
`,
		BackendConfig: fmt.Sprintf(`
mode http
server httpbin %s
`, backendAddr),
	}

	frontendSocket := cfg.Run(tb)

	return cfg, binURL, frontendSocket
}

// runCorazaRequestDetectOnly models full detect-only mode: HAProxy does NOT
// enforce the WAF verdict, so requests that the WAF would deny are still
// forwarded to the origin and their responses delivered. Both coraza-req and
// coraza-res carry detect-only=bool(true). The only deny left is the 504 on a
// SPOA processing error, so a failed request/response correlation (e.g. a
// "transaction not found" because an interrupted request was not cached)
// surfaces as a 504 instead of the expected 200.
func runCorazaRequestDetectOnly(tb testing.TB, directives string) (testutil.HAProxyConfig, string, string) {
	a, binURL, backendAddr := setupCorazaAgent(tb, directives)
	// Use an unknown app below to verify fallback keeps the configured metric label.

	// create the listener synchronously to prevent a race
	l := testutil.TCPListener(tb)
	// ignore errors as the listener will be closed by t.Cleanup
	go func() { _ = a.Serve(l) }()

	cfg := testutil.HAProxyConfig{
		EngineAddr:   l.Addr().String(),
		FrontendPort: fmt.Sprintf("%d", testutil.TCPPort(tb)),
		CustomFrontendConfig: `
    http-after-response set-header X-Anomaly-Score "%[var(txn.e2e.anomaly_score)]"
    http-after-response set-header X-Rules-Hit "%[var(txn.e2e.rules_hit)]"
    http-after-response set-header X-Rule-IDs "%[var(txn.e2e.rule_ids)]"

    # No is_deny enforcement: the WAF verdict is detected and logged but never
    # blocks, so every request reaches the origin and every response is returned.

    # Deny only when the SPOA itself errors, so a lost transaction surfaces.
    http-request deny deny_status 504 if { var(txn.e2e.error) -m int gt 0 }
    http-response deny deny_status 504 if { var(txn.e2e.error) -m int gt 0 }
`,
		EngineConfig: `
[e2e]
spoe-agent e2e
    messages    coraza-req     coraza-res
    option      var-prefix      e2e
    option      set-on-error    error
    timeout     hello           2s
    timeout     idle            2m
    timeout     processing      500ms
    use-backend e2e-spoa
    log         global

spoe-message coraza-req
    args app=str(unconfigured) src-ip=src src-port=src_port dst-ip=dst dst-port=dst_port method=method path=path query=query version=req.ver headers=req.hdrs body=req.body exportRuleIDs=bool(true) detect-only=bool(true)
    event on-frontend-http-request

spoe-message coraza-res
    args app=str(unconfigured) id=var(txn.e2e.id) version=res.ver status=status headers=res.hdrs body=res.body exportRuleIDs=bool(true) detect-only=bool(true)
    event on-http-response
`,
		BackendConfig: fmt.Sprintf(`
mode http
server httpbin %s
`, backendAddr),
	}

	frontendSocket := cfg.Run(tb)

	return cfg, binURL, frontendSocket
}

func runCorazaDetectOnly(tb testing.TB, directives string) (testutil.HAProxyConfig, string, string) {
	a, binURL, backendAddr := setupCorazaAgent(tb, directives)

	// create the listener synchronously to prevent a race
	l := testutil.TCPListener(tb)
	// ignore errors as the listener will be closed by t.Cleanup
	go func() { _ = a.Serve(l) }()

	cfg := testutil.HAProxyConfig{
		EngineAddr:   l.Addr().String(),
		FrontendPort: fmt.Sprintf("%d", testutil.TCPPort(tb)),
		CustomFrontendConfig: `
    # Currently haproxy cannot use variables to set the code or deny_status, so this needs to be manually configured here
    http-request redirect code 302 location %[var(txn.e2e.data)] if { var(txn.e2e.action) -m str redirect }
    http-response redirect code 302 location %[var(txn.e2e.data)] if { var(txn.e2e.action) -m str redirect }

    acl is_deny var(txn.e2e.action) -m str deny
    acl status_424 var(txn.e2e.status) -m int 424

    http-after-response set-header X-Anomaly-Score "%[var(txn.e2e.anomaly_score)]"
    http-after-response set-header X-Rules-Hit "%[var(txn.e2e.rules_hit)]"
    http-after-response set-header X-Rule-IDs "%[var(txn.e2e.rule_ids)]"

    # Special check for e2e tests as they validate the config.
    http-request deny deny_status 424 hdr waf-block "request" if is_deny status_424
    http-response deny deny_status 424 hdr waf-block "response" if is_deny status_424

    http-request deny deny_status 403 hdr waf-block "request" if is_deny
    http-response deny deny_status 403 hdr waf-block "response" if is_deny

    http-request silent-drop if { var(txn.e2e.action) -m str drop }
    http-response silent-drop if { var(txn.e2e.action) -m str drop }

    # Deny in case of an error, when processing with the Coraza SPOA
    http-request deny deny_status 504 if { var(txn.e2e.error) -m int gt 0 }
    http-response deny deny_status 504 if { var(txn.e2e.error) -m int gt 0 }
`,
		EngineConfig: `
[e2e]
spoe-agent e2e
    messages    coraza-req     coraza-res
    option      var-prefix      e2e
    option      set-on-error    error
    timeout     hello           2s
    timeout     idle            2m
    timeout     processing      500ms
    use-backend e2e-spoa
    log         global

spoe-message coraza-req
    args app=str(default) src-ip=src src-port=src_port dst-ip=dst dst-port=dst_port method=method path=path query=query version=req.ver headers=req.hdrs body=req.body exportRuleIDs=bool(true)
    event on-frontend-http-request

spoe-message coraza-res
    args app=str(default) id=var(txn.e2e.id) version=res.ver status=status headers=res.hdrs body=res.body exportRuleIDs=bool(true) detect-only=bool(true)
    event on-http-response
`,
		BackendConfig: fmt.Sprintf(`
mode http
server httpbin %s
`, backendAddr),
	}

	frontendSocket := cfg.Run(tb)

	return cfg, binURL, frontendSocket
}

// Snapshot the exported metrics around the existing HTTP requests. These tests
// run serially because the production collectors use the default registry.
type requestMetrics struct {
	requests, transactions, verdicts, scoreCount, scoreSum, suspicious float64
	rules                                                              map[string]float64
	durations                                                          map[string]float64
}

func gatherRequestMetrics(t *testing.T, outcome, mode string) requestMetrics {
	t.Helper()
	families, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatal(err)
	}
	result := requestMetrics{rules: make(map[string]float64), durations: make(map[string]float64)}
	for _, family := range families {
		for _, metric := range family.Metric {
			labels := make(map[string]string)
			for _, label := range metric.Label {
				labels[label.GetName()] = label.GetValue()
			}
			// Only count the configured test application. Missing labels and
			// labels derived from the incoming fallback name fail the delta checks.
			if labels["application"] != "default" {
				continue
			}
			switch family.GetName() {
			case "coraza_handle_spoe_duration_seconds":
				result.durations[labels["phase"]+"/"+labels["result"]] += float64(metric.GetHistogram().GetSampleCount())
			case "coraza_requests_total":
				result.requests += metric.GetCounter().GetValue()
			case "coraza_transactions_total":
				if labels["suspicious"] != "true" && labels["suspicious"] != "false" {
					t.Error("transaction metric has invalid suspicious label")
				}
				if labels["suspicious"] == "true" {
					result.suspicious += metric.GetCounter().GetValue()
				}
				result.transactions += metric.GetCounter().GetValue()
				if labels["outcome"] == outcome && labels["mode"] == mode {
					result.verdicts += metric.GetCounter().GetValue()
				}
			case "coraza_rule_matches_total":
				if labels["severity"] == "" {
					t.Error("rule match metric has no severity")
				}
				result.rules[labels["rule_id"]] += metric.GetCounter().GetValue()
			case "coraza_inbound_anomaly_score":
				result.scoreCount += float64(metric.GetHistogram().GetSampleCount())
				result.scoreSum += metric.GetHistogram().GetSampleSum()
			}
		}
	}
	return result
}

func checkRequestMetrics(t *testing.T, outcome, mode string, suspicious bool) func(*http.Response) {
	t.Helper()
	before := gatherRequestMetrics(t, outcome, mode)
	return func(resp *http.Response) {
		t.Helper()
		score, err := strconv.ParseFloat(resp.Header.Get("X-Anomaly-Score"), 64)
		if err != nil {
			t.Fatalf("invalid anomaly score header: %v", err)
		}
		expectedRules := make(map[string]float64)
		if ids := resp.Header.Get("X-Rule-IDs"); ids != "" {
			for _, id := range strings.Split(ids, ",") {
				expectedRules[id]++
			}
		}
		expectedDurations := map[string]float64{"request/success": 1}
		if outcome == "deny" {
			expectedDurations = map[string]float64{"request/interrupted": 1}
		}
		if outcome == "allow" || mode == "detect_only" {
			expectedDurations["response/success"] = 1
		}
		var messageCount float64
		for _, count := range expectedDurations {
			messageCount += count
		}

		var after requestMetrics
		// Detect-only evaluation completes after HAProxy receives the SPOE reply.
		if !pollUntil(time.Now().Add(5*time.Second), time.Millisecond, func() bool {
			after = gatherRequestMetrics(t, outcome, mode)
			var durationCount float64
			for key, count := range after.durations {
				durationCount += count - before.durations[key]
			}
			return after.scoreCount-before.scoreCount >= 1 && after.transactions-before.transactions >= 1 && durationCount >= messageCount
		}) {
			t.Fatal("transaction metrics did not complete")
		}
		// Gather again after completion so concurrent collector reads cannot mix
		// values from before and after the background evaluation.
		after = gatherRequestMetrics(t, outcome, mode)
		for name, delta := range map[string]float64{
			"requests":           after.requests - before.requests,
			"transactions":       after.transactions - before.transactions,
			"verdicts":           after.verdicts - before.verdicts,
			"score observations": after.scoreCount - before.scoreCount,
		} {
			if delta != 1 {
				t.Errorf("expected one %s increment, got %v", name, delta)
			}
		}
		for key := range after.durations {
			if delta := after.durations[key] - before.durations[key]; delta != expectedDurations[key] {
				t.Errorf("duration observations for %s: got %v, want %v", key, delta, expectedDurations[key])
			}
		}

		if delta := after.scoreSum - before.scoreSum; delta != score {
			t.Errorf("score sum increased by %v, want %v", delta, score)
		}
		// Prometheus also includes rules deliberately excluded from HAProxy's
		// attack-only variables, including this message-less custom rule.
		expectedRules["1234567"] = 0
		if resp.Request.Header.Get("coraza-e2e") == "ok" {
			expectedRules["1234567"] = 1
		}
		for id, want := range expectedRules {
			if delta := after.rules[id] - before.rules[id]; delta != want {
				t.Errorf("rule %s increased by %v, want %v", id, delta, want)
			}
		}
		wantSuspicious := float64(0)
		if suspicious {
			wantSuspicious = 1
		}
		if got := after.suspicious - before.suspicious; got != wantSuspicious {
			t.Errorf("suspicious completions: got %v, want %v", got, wantSuspicious)
		}
	}
}
