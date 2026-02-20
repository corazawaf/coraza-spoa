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
coraza-spoa -config /etc/coraza-spoa/config.yaml
```

Note: The example file is named `coraza-spoa.yaml`, but it's recommended to rename it to `config.yaml` when deploying, especially if using the systemd service file from `contrib/coraza-spoa.service`.

## HAProxy SPOE

Configure HAProxy to exchange messages with the SPOA. The example SPOE configuration file is [coraza.cfg](https://github.com/corazawaf/coraza-spoa/blob/main/example/haproxy/coraza.cfg), you can copy it and modify the related configuration information. Default directory to place the config is `/etc/haproxy/coraza.cfg`.

```ini
# /etc/haproxy/coraza.cfg
[coraza]
spoe-agent coraza-agent
    groups      coraza-req
    option      var-prefix      coraza
    option      set-on-error    error
    timeout     hello           2s
    timeout     idle            2m
    timeout     processing      500ms
    use-backend coraza-spoa
    log         global

spoe-message coraza-req
    # Arguments are required to be in this order
    args app=var(txn.coraza.app) src-ip=src src-port=src_port dst-ip=dst dst-port=dst_port method=method path=path query=query version=req.ver headers=req.hdrs body=req.body

spoe-group coraza-req
    messages coraza-req
```

The application name from `config.yaml` must match the `app` variable set in the HAProxy configuration. Note that the `app` argument uses `var(txn.coraza.app)` which is set in the HAProxy frontend configuration (see below).

The backend defined in `use-backend` must match a `haproxy.cfg` backend which directs requests to the SPOA daemon reachable via `127.0.0.1:9000`.

## HAProxy

Configure HAProxy with a frontend, which sets the coraza app variable, configures the SPOE filter, sends requests to the SPOE for processing, and handles the returned actions. Also add a backend section, which is referenced by use-backend in `coraza.cfg`.

```haproxy
# /etc/haproxy/haproxy.cfg
frontend web
    # Set coraza app variable - must match application name in coraza-spoa.yaml
    http-request set-var(txn.coraza.app) str(sample_app)

    # Configure SPOE filter and send requests for processing
    filter spoe engine coraza config /etc/haproxy/coraza.cfg
    http-request send-spoe-group coraza coraza-req

    # Handle redirect action
    http-request redirect code 302 location %[var(txn.coraza.data)] if { var(txn.coraza.action) -m str redirect }
    http-response redirect code 302 location %[var(txn.coraza.data)] if { var(txn.coraza.action) -m str redirect }

    # Handle deny action
    http-request deny deny_status 403 hdr waf-block "request"  if { var(txn.coraza.action) -m str deny }
    http-response deny deny_status 403 hdr waf-block "response" if { var(txn.coraza.action) -m str deny }

    # Handle drop action
    http-request silent-drop if { var(txn.coraza.action) -m str drop }
    http-response silent-drop if { var(txn.coraza.action) -m str drop }

    # Handle SPOA processing errors
    http-request deny deny_status 500 if { var(txn.coraza.error) -m int gt 0 }
    http-response deny deny_status 500 if { var(txn.coraza.error) -m int gt 0 }

    ...

backend coraza-spoa
    option spop-check
    mode tcp
    server s1 127.0.0.1:9000 check
```

A comprehensive HAProxy configuration example can be found in [example/haproxy/haproxy.cfg](https://github.com/corazawaf/coraza-spoa/blob/main/example/haproxy/haproxy.cfg).

In the SPOE configuration file (coraza.cfg), we declare the [coraza-spoa backend](https://github.com/corazawaf/coraza-spoa/blob/main/example/haproxy/coraza.cfg#L13) to communicate with the service, so we also need to define it in the [HAProxy file](https://github.com/corazawaf/coraza-spoa/blob/main/example/haproxy/haproxy.cfg#L54).

The `http-request set-var(txn.coraza.app)` directive sets the application name that will be used by the SPOA to determine which Coraza configuration to apply. This should match one of the application names defined in your `coraza-spoa.yaml` configuration file. You can customize this per virtual host or use HAProxy variables such as `fe_name` (frontend name) instead of a hardcoded string.

## Systemd Service

If you intend to access the coraza-spoa service from another machine (non-localhost), you need to modify the network binding directives in [contrib/coraza-spoa.service](https://github.com/corazawaf/coraza-spoa/blob/main/contrib/coraza-spoa.service). By default, the service restricts access to localhost only:

```ini
IPAddressDeny=any
IPAddressAllow=localhost
```

To allow access from other machines, update the `IPAddressAllow` directive to include the appropriate IP addresses or network ranges (e.g., `IPAddressAllow=localhost 192.168.1.0/24`).

## Docker

- Build the coraza-spoa image `cd ./example ; docker compose build`
- Run haproxy, coraza-spoa and a mock server `docker compose up`
- Perform a request which gets blocked by the WAF: `curl http://localhost:8080/\?x\=/etc/passwd`

## Kubernetes

For deploying Coraza SPOA on Kubernetes, you can use the official Helm chart available at [corazawaf/charts](https://github.com/corazawaf/charts/tree/main/charts/coraza-spoa).
