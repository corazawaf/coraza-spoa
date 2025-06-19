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

The command `go run mage.go build` will compile the source code and produce the executable file `coraza-spoa`.

## Configuration

## Coraza SPOA

The example configuration file is [example/coraza-spoa.yaml](https://github.com/corazawaf/coraza-spoa/blob/main/example/coraza-spoa.yaml), you can copy it and modify the related configuration information. You can start the service by running the command:

```
coraza-spoa -f /etc/coraza-spoa/coraza-spoa.yaml
```

You will also want to download & extract the [OWASP Core Ruleset]( https://github.com/coreruleset/coreruleset/releases) (version 4+ supported) to the `/etc/coraza-spoa` directory.

## HAProxy SPOE

Configure HAProxy to exchange messages with the SPOA. The example SPOE configuration file is [coraza.cfg](https://github.com/corazawaf/coraza-spoa/blob/main/example/haproxy/coraza.cfg), you can copy it and modify the related configuration information. Default directory to place the config is `/etc/haproxy/coraza.cfg`.

```ini
# /etc/haproxy/coraza.cfg
spoe-agent coraza-agent
    ...
    use-backend coraza-spoa

spoe-message coraza-req
    args app=str(sample_app) id=unique-id src-ip=src ...
    event on-frontend-http-request
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
    http-request deny deny_status 403 hdr waf-block "request"  if { var(txn.coraza.action) -m str deny }
    ...

backend coraza-spoa
    mode tcp
    server s1 127.0.0.1:9000
```

A comprehensive HAProxy configuration example can be found in [example/haproxy/coraza.cfg](https://github.com/corazawaf/coraza-spoa/blob/main/example/haproxy/coraza.cfg).

Because, in the SPOE configuration file (coraza.cfg), we declare to use the backend [coraza-spoa](https://github.com/corazawaf/coraza-spoa/blob/main/example/haproxy/coraza.cfg#L14) to communicate with the service, so we need also to define it in the [HAProxy file](https://github.com/corazawaf/coraza-spoa/blob/main/example/haproxy/haproxy.cfg#L37):

If you intend to access coraza-spoa service from another machine, remember to change the binding networking directives (IPAddressAllow/IPAddressDeny) in [contrib/coraza-spoa.service](https://github.com/corazawaf/coraza-spoa/blob/main/contrib/coraza-spoa.service)

## Docker

- Build the coraza-spoa image `cd ./example ; docker compose build`
- Run haproxy, coraza-spoa and a mock server `docker compose up`
- Perform a request which gets blocked by the WAF: `curl http://localhost:8080/\?x\=/etc/passwd`
