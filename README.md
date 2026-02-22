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
    ...
    use-backend coraza-spoa

spoe-message coraza-req
    args app=str(sample_app) id=unique-id src-ip=src ...
```

The application name from `config.yaml` must match the `app=` name.

The backend defined in `use-backend` must match a `haproxy.cfg` backend which directs requests to the SPOA daemon reachable via `127.0.0.1:9000`.

Instead of the hard coded application name `str(sample_app)` you can use some HAProxy variables. For example, frontend name `fe_name`.

## HAProxy

Configure HAProxy with a frontend, which contains a `filter` statement to forward requests to the SPOA and deny based on the returned action. Also add a backend section, which is referenced by use-backend in `coraza.cfg`.

```haproxy
# /etc/haproxy/haproxy.cfg
frontend web
    filter spoe engine coraza config /etc/haproxy/coraza.cfg
    ...
    http-request deny deny_status 403 hdr waf-block "request" if { var(txn.coraza.action) -m str deny }
    ...

backend coraza-spoa
    mode tcp
    option spop-check
    server s1 127.0.0.1:9000 check
```

A comprehensive HAProxy configuration example can be found in [example/haproxy/coraza.cfg](https://github.com/corazawaf/coraza-spoa/blob/main/example/haproxy/coraza.cfg).

Because, in the SPOE configuration file (coraza.cfg), we declare to use the backend [coraza-spoa](https://github.com/corazawaf/coraza-spoa/blob/main/example/haproxy/coraza.cfg#L13) to communicate with the service, so we need also to define it in the [HAProxy file](https://github.com/corazawaf/coraza-spoa/blob/main/example/haproxy/haproxy.cfg#L50):

If you intend to access coraza-spoa service from another machine, remember to change the binding networking directives (IPAddressAllow/IPAddressDeny) in [contrib/coraza-spoa.service](https://github.com/corazawaf/coraza-spoa/blob/main/contrib/coraza-spoa.service)

## HAProxy Logging

To gain full visibility into WAF actions directly from your HAProxy logs, you can use the transaction variables exported by the Coraza-SPOA agent.

### Available Variables

The agent populates the following variables in the `txn` scope:

* **`txn.coraza.id`**: The unique transaction ID.
* **`txn.coraza.status`**: The HTTP status code determined by the WAF (e.g., 403).
* **`txn.coraza.anomaly_score`**: The total inbound anomaly score for the request.
* **`txn.coraza.rules_hit`**: The total count of triggered attack rules.
* **`txn.coraza.rule_ids`**: A comma-separated list of triggered Rule IDs.
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
```coraza-spoa.yaml
  SecAction "id:100010,phase:1,nolog,pass,setvar:tx.spoa_export_rule_ids=1"
  Include @coraza.conf
  .....
```
```haproxy
log-format "%ci:%cp\ [%t]\ %ft\ %b/%s\ %Th/%Ti/%TR/%Tq/%Tw/%Tc/%Tr/%Tt\ %ST\ %B\ %CC\ %CS\ %tsc\ %ac/%fc/%bc/%sc/%rc\ %sq/%bq\ %hr\ %hs\ %{+Q}r\ %[var(txn.coraza.id)]\ spoa-error:\ %[var(txn.coraza.error)]\ waf-hit:\ %[var(txn.coraza.status)]\ rule_ids:\ %[var(txn.coraza.rule_ids)]\ rules-hit:\ %[var(txn.coraza.rules_hit)]"
```

### Custom Rules & ID Ranges Allocation

To avoid conflicts with the OWASP Core Rule Set (CRS) and to ensure that the SPOA agent exports accurate metrics to HAProxy (`rules_hit` & `rule_ids`), you must strictly adhere to the following Rule ID ranges for local rules:

* **Infrastructure & Whitelists (IDs: 100000 - 189999):** Use this range for IP whitelists, disabling specific CRS rules, or tuning (e.g., GeoIP limits). Rules in this range are **intentionally ignored** by the SPOA agent's attack counter to prevent false positives in your HAProxy metrics.
* **Custom Attack & Hardening Rules (IDs: 190000 - 199999):** Use this range for actual security blocks and custom hardening rules. Rules in this range are actively monitored. If triggered, they will increment the `rules_hit` counter and their IDs will be exported in the `rule_ids` variable.

#### Example: WordPress Hardening Rule

Below is an example of a custom hardening rule designed to protect WordPress environments by preventing the direct execution of internal PHP scripts inside the `wp-includes` and `wp-content/uploads` directories. 

**Crucial Placement Rule:** This rule (and any custom hardening rule) **MUST be placed AFTER the inclusion of the main CRS rules** (e.g., after `Include /etc/coraza-spoa/owasp-crs/rules/*.conf` or your specific CRS path). Placing it here ensures that the anomaly scoring variables (like `tx.inbound_anomaly_score_pl1`) are properly initialized by the CRS before your custom rule attempts to add points to them.

```yaml
      # HARDENING BLOCK: (Adds 15 points to the previously initialized anomaly score bucket)
      # Example: WordPress protection against direct internal PHP execution
      #SecRule REQUEST_URI "@rx ^/(?:wp-includes|wp-content/uploads)/.*\.php$" \
      #  "id:190500, \
      #  phase:1, \
      #  pass, \
      #  log, \
      #  msg:'HARDENING BLOCK: Direct execution of internal PHP script attempted', \
      #  tag:'wordpress_hardening', \
      #  setvar:'tx.inbound_anomaly_score_pl1=+15'"
```

## Docker

- Build the coraza-spoa image `cd ./example ; docker compose build`
- Run haproxy, coraza-spoa and a mock server `docker compose up`
- Perform a request which gets blocked by the WAF: `curl http://localhost:8080/\?x\=/etc/passwd`
