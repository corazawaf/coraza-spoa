<h1>
  <img src="https://coraza.io/images/logo_shield_only.png" align="left" height="46px" alt=""/>
  <span>Coraza SPOA - HAProxy Web Application Firewall</span>
</h1>

[![Code Linting](https://github.com/corazawaf/coraza-spoa/actions/workflows/lint.yaml/badge.svg)](https://github.com/corazawaf/coraza-spoa/actions/workflows/lint.yaml)
[![CodeQL Scanning](https://github.com/corazawaf/coraza-spoa/actions/workflows/codeql.yaml/badge.svg)](https://github.com/corazawaf/coraza-spoa/actions/workflows/codeql.yaml)

Coraza SPOA is a system daemon which brings the Coraza Web Application Firewall (WAF) as a backing service for HAProxy. It is written in Go, Coraza supports ModSecurity SecLang rulesets and is 100% compatible with the OWASP Core Rule Set v4.

HAProxy includes a [Stream Processing Offload Engine](https://www.haproxy.com/blog/extending-haproxy-with-the-stream-processing-offload-engine) [SPOE](https://raw.githubusercontent.com/haproxy/haproxy/master/doc/SPOE.txt) to offload request processing to a Stream Processing Offload Agent (SPOA). Coraza SPOA embeds the [Coraza Engine](https://github.com/corazawaf/coraza), loads the ruleset and filters http requests or application responses which are passed forwarded by HAProxy for inspection.

## Compilation

### Build

The command `go run mage.go build` will compile the source code and produce the executable file `coraza-spoa` inside the `build/` folder.

## Configuration

## Coraza SPOA

The example configuration file is [example/coraza-spoa.yaml](https://github.com/corazawaf/coraza-spoa/blob/main/example/coraza-spoa.yaml), you can copy it and modify the related configuration information. You can start the service by running the command:

```
coraza-spoa -config /etc/coraza-spoa/coraza-spoa.yaml
```

## HAProxy SPOE

Configure HAProxy to exchange messages with the SPOA. The example SPOE configuration file is [coraza.cfg](https://github.com/corazawaf/coraza-spoa/blob/main/example/haproxy/coraza.cfg), you can copy it and modify the related configuration information. Default directory to place the config is `/etc/haproxy/coraza.cfg`.

```ini
# /etc/haproxy/coraza.cfg
spoe-agent coraza-agent
    groups      coraza-req
    ...
    use-backend coraza-spoa

spoe-message coraza-req
    args app=var(txn.coraza.app) src-ip=src ...

spoe-group coraza-req
    messages coraza-req
```

The application name from `config.yaml` must match the `app` variable set in the HAProxy configuration (see below).

The backend defined in `use-backend` must match a `haproxy.cfg` backend which directs requests to the SPOA daemon reachable via `127.0.0.1:9000`.

## HAProxy

Configure HAProxy with a frontend, which contains a `filter` statement to forward requests to the SPOA and deny based on the returned action. Also add a backend section, which is referenced by use-backend in `coraza.cfg`.

```haproxy
# /etc/haproxy/haproxy.cfg
frontend web
    # Set application name variable for SPOA
    http-request set-var(txn.coraza.app) str(sample_app)
    
    filter spoe engine coraza config /etc/haproxy/coraza.cfg
    http-request send-spoe-group coraza coraza-req
    ...
    http-request deny deny_status 403 hdr waf-block "request" if { var(txn.coraza.action) -m str deny }
    ...

backend coraza-spoa
    mode tcp
    option spop-check
    server s1 127.0.0.1:9000 check
```

A comprehensive HAProxy configuration example can be found in [example/haproxy/haproxy.cfg](https://github.com/corazawaf/coraza-spoa/blob/main/example/haproxy/haproxy.cfg).

In the SPOE configuration file (coraza.cfg), we declare the [coraza-spoa backend](https://github.com/corazawaf/coraza-spoa/blob/main/example/haproxy/coraza.cfg#L13) to communicate with the service, so we also need to define it in the [HAProxy file](https://github.com/corazawaf/coraza-spoa/blob/main/example/haproxy/haproxy.cfg#L54).

**Note:** It is recommended to run coraza-spoa on the same host as HAProxy to minimize latency. The [systemd service file](https://github.com/corazawaf/coraza-spoa/blob/main/contrib/coraza-spoa.service) restricts network access to localhost only by default for security.

## HAProxy Logging

To gain full visibility into WAF actions directly from your HAProxy logs, you can use the transaction variables exported by the Coraza-SPOA agent.

### Available Variables

The agent populates the following variables in the `txn` scope:

* **`txn.coraza.id`**: The unique transaction ID.
* **`txn.coraza.status`**: The HTTP status code determined by the WAF (e.g., 403).
* **`txn.coraza.anomaly_score`**: The total inbound anomaly score for the request.
* **`txn.coraza.rules_hit`**: The total count of triggered attack rules.
* **`txn.coraza.rule_ids`**: A comma-separated list of triggered Rule IDs (if enabled).
* **`txn.coraza.error`**: Contains SPOA-related errors if the transaction fails.

### Example Log Formats

You can incorporate these variables into your `log-format` directive in `haproxy.cfg`.

**1. Standard Score Tracking**
Use this for general monitoring of threat levels and rule counts:

```haproxy
log-format "%ci:%cp\ [%t]\ %ft\ %b/%s\ %Th/%Ti/%TR/%Tq/%Tw/%Tc/%Tr/%Tt\ %ST\ %B\ %CC\ %CS\ %tsc\ %ac/%fc/%bc/%sc/%rc\ %sq/%bq\ %hr\ %hs\ %{+Q}r\ %[var(txn.coraza.id)]\ spoa-error:\ %[var(txn.coraza.error)]\ waf-hit:\ %[var(txn.coraza.status)]\ score:%[var(txn.coraza.anomaly_score)]\ rules_hit:%[var(txn.coraza.rules_hit)]"
```

**2. Extended Debugging (with Rule IDs)**
Use this if you need to identify exactly which rules were triggered to troubleshoot false positives. 

> **Note:** Exporting the specific Rule IDs requires explicit activation in your Coraza configuration.
```coraza.cfg
spoe-message coraza-req
    
    args app= ... exportRuleIDs=bool(true)

spoe-message coraza-res
    
    args app= ... exportRuleIDs=bool(true)

  .....
```
```haproxy
log-format "%ci:%cp\ [%t]\ %ft\ %b/%s\ %Th/%Ti/%TR/%Tq/%Tw/%Tc/%Tr/%Tt\ %ST\ %B\ %CC\ %CS\ %tsc\ %ac/%fc/%bc/%sc/%rc\ %sq/%bq\ %hr\ %hs\ %{+Q}r\ %[var(txn.coraza.id)]\ spoa-error:\ %[var(txn.coraza.error)]\ waf-hit:\ %[var(txn.coraza.status)]\ rule_ids:\ %[var(txn.coraza.rule_ids)]\ rules-hit:\ %[var(txn.coraza.rules_hit)]"
```

### Custom Rules & ID Ranges Allocation

To avoid conflicts with the OWASP Core Rule Set (CRS) and to ensure that the SPOA agent exports accurate metrics to HAProxy (`rules_hit` & `rule_ids`), you must strictly adhere to the following Rule ID ranges for local rules:

* **Infrastructure & Whitelists (IDs: 100000 - 189999):** Use this range for IP whitelists, disabling specific CRS rules, or tuning (e.g., GeoIP limits). Rules in this range are **intentionally ignored** by the SPOA agent's attack counter to prevent false positives in your HAProxy metrics.
* **Custom Attack & Hardening Rules (IDs: 190000 - 199999):** Use this range for actual security blocks and custom hardening rules. Rules in this range are actively monitored. If triggered, they will increment the `rules_hit` counter and their IDs will be exported in the `rule_ids` variable.

## Docker

- Build the coraza-spoa image `cd ./example ; docker compose build`
- Run haproxy, coraza-spoa and a mock server `docker compose up`
- Perform a request which gets blocked by the WAF: `curl http://localhost:8080/\?x\=/etc/passwd`

## Kubernetes

For deploying Coraza SPOA on Kubernetes, you can use the official Helm chart available at [corazawaf/charts](https://github.com/corazawaf/charts/tree/main/charts/coraza-spoa).

### Prometheus metrics

Enable the `/metrics` endpoint with `-metrics-addr=:9000`. Counters reset when
SPOA restarts; use `rate()` or `increase()` in PromQL.

| Metric | Type | Meaning |
| --- | --- | --- |
| `coraza_handle_spoe_duration_seconds{application,phase,result}` | Histogram | Time spent handling each SPOE message, including unknown messages. In detect-only response mode this measures dispatch, not background evaluation. |
| `coraza_requests_total{application}` | Counter | HTTP requests received for WAF evaluation (one per transaction created). |
| `coraza_transactions_total{application,mode,outcome,suspicious}` | Counter | Transactions finished after request evaluation, response evaluation, or expiry. |
| `coraza_rule_matches_total{application,rule_id,severity}` | Counter | All matched rules recorded once at transaction completion, including custom IDs outside the attack ranges and rules without messages. |
| `coraza_inbound_anomaly_score{application}` | Histogram | Final `blocking_inbound_anomaly_score`, when present and a valid nonnegative integer. Missing scores are not recorded as zero. |
| `coraza_ruleset_info{application,ruleset,version}` | Gauge | Constant 1 for each ruleset version observed while loading the active application configuration, including included files. |

The `suspicious` label is `true` for completed transactions with no interruption
or evaluation error and a positive inbound anomaly score below their configured
CRS threshold. All other completions use `false`, including missing or invalid
scores/thresholds; `false` does not necessarily mean a clean request. Summing
across this label gives the total without counting any transaction twice.

The `application` label uses the configured application name, including when an
unknown SPOE app falls back to the default application. Request counts, transaction
completions, rule matches, anomaly scores, ruleset information, and SPOE duration
all use this label. SPOE duration uses an empty application label if handling
fails before an application is resolved, or the message is unknown.

For SPOE duration, `phase` is `request`, `response`, or `unknown`, and `result`
is `success`, `interrupted`, `error`, or `unknown_message`. These describe the
handler call, not the final transaction outcome. An asynchronous detect-only
response records dispatch success even if later evaluation interrupts or fails.
Arbitrary incoming message names are never used as label values.

Transaction `outcome` is `allow`, `deny`, `drop`, `redirect`, `interrupted`
(other disruptive actions), `error`, or `expired` (response never arrived).
These describe WAF evaluation, not the HTTP status or what HAProxy enforced.
The `mode` label is `enforce` or `detect_only`, taken from the request's SPOE
flag; it does not describe the rules' `SecRuleEngine` setting. Detect-only
interruptions remain correlated through response evaluation and are counted
once, including when evaluation runs in the background. Expired transactions
contribute their available rule matches and scores, but are not considered
successful or suspicious completions.

Ruleset metadata comes from each rule's `ver` action at configuration load time,
without waiting for a match. `OWASP_CRS/4.25.0` becomes `ruleset="OWASP_CRS"`,
`version="4.25.0"`; splitting uses the last slash. Unqualified versions use an
empty `ruleset` label, and rules without `ver` contribute no version metadata.
Activating a replacement removes metadata from the previous configuration;
failed or merely prepared configurations do not affect the metric. This reports
versions encountered during loading, including rules subsequently disabled by
configuration, rather than a count of enabled rules. HAProxy's `rules_hit` and
`rule_ids` retain their existing attack-range and nonempty-message filters.

Rule IDs come from configured rules and severity uses Coraza's fixed vocabulary.
No request paths, transaction IDs, client addresses, or incoming application
names are used as labels. Separate totals by rule or severity can be calculated
from the same counter:

```promql
# HTTP request rate by application
sum by (application) (rate(coraza_requests_total[5m]))

# Rule matches per second by application and severity
sum by (application, severity) (rate(coraza_rule_matches_total[5m]))

# WAF verdicts per second, preserving detect-only mode
sum by (outcome, mode) (rate(coraza_transactions_total[5m]))

# Suspicious completions per second by application
sum by (application) (rate(coraza_transactions_total{suspicious="true"}[5m]))

# 95th percentile SPOE handling duration by application and phase
histogram_quantile(0.95,
  sum by (application, phase, le) (rate(coraza_handle_spoe_duration_seconds_bucket[5m])))

# Average inbound anomaly score by application
sum by (application) (rate(coraza_inbound_anomaly_score_sum[5m]))
  / sum by (application) (rate(coraza_inbound_anomaly_score_count[5m]))
```

Request and completion rates can differ while responses are pending. The SPOE
duration histogram's `_count` counts messages, so it is not an HTTP request total.
