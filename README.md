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

## Prometheus Metrics

When started with `--metrics-addr=<host>:<port>`, Coraza SPOA exposes Prometheus metrics at `/metrics`. The following series are available:

| Metric | Type | Labels | Description |
|---|---|---|---|
| `coraza_handle_spoe_duration_seconds` | Histogram | - | Wall-clock duration of each SPOE message handler call, using the default Prometheus buckets. |
| `coraza_actions_total` | CounterVec | `action`, `application` | WAF verdicts per request. `action` is `allow` when no rule interrupted, otherwise the interruption action (`deny`, `drop`, `redirect`, etc). `application` is the requested SPOE `app` arg when it matches a configured application, or the `default_application`'s name when fallback handles an unknown name - keeping the label bounded to `applications[].name` even if the SPOE `app` arg is sourced from request data (e.g. `hdr(host)`). The unmatched requested name is still logged at debug level. Counted exactly once per request - at the response phase when `ResponseCheck` is enabled, or at the request phase otherwise. |
| `coraza_rule_triggers_total` | CounterVec | `rule_id`, `severity` | One increment per matched attack-range rule (CRS `910000-959999` and local `190000-199999`). Rules outside these ranges (whitelists, tuning) are intentionally excluded to keep cardinality bounded. `severity` is one of `emergency`, `alert`, `critical`, `error`, `warning`, `notice`, `info`, `debug`, `unknown`. |
| `coraza_anomaly_score` | Histogram | - | Distribution of the CRS `tx.blocking_inbound_anomaly_score` observed at the end of each transaction. Buckets are tuned for CRS scoring: `0, 3, 5, 7, 10, 15, 25, 50, 100`. The `5` boundary corresponds to the default CRS deny threshold. |

`coraza_actions_total`, `coraza_rule_triggers_total`, and `coraza_anomaly_score` are observed exactly once per transaction even when `ResponseCheck` is enabled (the response phase is treated as the final verdict; the request phase is treated as final only on interruption or when `ResponseCheck` is off).

When `ResponseCheck` is enabled but HAProxy never fires `on-http-response` for a request (e.g. backend timeout, client disconnect), the transaction sits in the response-cache until `transaction_ttl_ms` expires. `coraza_rule_triggers_total` and `coraza_anomaly_score` are still recorded from the request-phase observations when the cache evicts the orphaned transaction; `coraza_actions_total` is not - an orphaned transaction never produced a verdict, so it is excluded rather than mislabelled as `allow`.

### Example queries

```promql
# Deny rate (denials per second) per application.
sum by (application) (rate(coraza_actions_total{action="deny"}[5m]))

# Deny ratio per application.
sum by (application) (rate(coraza_actions_total{action="deny"}[5m]))
  / sum by (application) (rate(coraza_actions_total[5m]))

# Top 10 most-triggered attack rules.
topk(10, sum by (rule_id) (rate(coraza_rule_triggers_total[15m])))

# 95th percentile anomaly score over the last 15 minutes.
histogram_quantile(0.95, sum by (le) (rate(coraza_anomaly_score_bucket[15m])))
```

## Docker

- Build the coraza-spoa image `cd ./example ; docker compose build`
- Run haproxy, coraza-spoa and a mock server `docker compose up`
- Perform a request which gets blocked by the WAF: `curl http://localhost:8080/\?x\=/etc/passwd`

## Kubernetes

For deploying Coraza SPOA on Kubernetes, you can use the official Helm chart available at [corazawaf/charts](https://github.com/corazawaf/charts/tree/main/charts/coraza-spoa).
