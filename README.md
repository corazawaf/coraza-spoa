# Coraza SPOA HAProxy Web Application Firewall

[![Code Linting](https://github.com/corazawaf/coraza-spoa/actions/workflows/lint.yaml/badge.svg)](https://github.com/corazawaf/coraza-spoa/actions/workflows/lint.yaml)
[![CodeQL Scanning](https://github.com/corazawaf/coraza-spoa/actions/workflows/codeql.yaml/badge.svg)](https://github.com/corazawaf/coraza-spoa/actions/workflows/codeql.yaml)

## Overview

Coraza SPOA is a system daemon which runs the Coraza Web Application Firewall (WAF) as a backing service for HAProxy.  HAProxy includes a [Stream Processing Offload Engine] (https://www.haproxy.com/blog/extending-haproxy-with-the-stream-processing-offload-engine) [SPOE](https://raw.githubusercontent.com/haproxy/haproxy/master/doc/SPOE.txt) to offload request processing to a Stream Processing Offload Agent (SPOA). The SPOA analyzes the request and response using [OWASP Coraza](https://github.com/corazawaf/coraza) and provides the final verdict back to HAProxy.

## Compilation

### Build

The command `make` will compile the source code and produce the executable file `coraza-spoa`.

### Clean

When you need to re-compile the source code, you can use the command `make clean` to clean the executable file.

## Coraza SPOA

The example configuration file is [config.yaml.default](https://github.com/corazawaf/coraza-spoa/blob/main/config.yaml.default), you can copy it and modify the related configuration information. You can start the service by running the command:

```
coraza-spoa -config /etc/coraza-spoa/coraza.yaml
```


## Configure a SPOE to use the service

The example SPOE configuration file is [coraza.cfg](https://github.com/corazawaf/coraza-spoa/blob/main/doc/config/coraza.cfg), you can copy it and modify the related configuration information. Default directory to place the config is `/etc/haproxy/coraza.cfg`.

```ini
# /etc/haproxy/coraza.cfg
[coraza]
spoe-agent coraza-agent
    # Filter http requests (the response is not evaluated)
    messages    coraza-req
    # Comment the previous line and add coraza-res, to also apply response filters.
    # NOTE: there are still some memory & caching issues, so use this with care
    #messages   coraza-req     coraza-res
    option      var-prefix      coraza
    option      set-on-error    error
    timeout     hello           2s
    timeout     idle            2m
    timeout     processing      500ms
    use-backend coraza-spoa
    log global

spoe-message coraza-req
    args app=str(sample_app) id=unique-id src-ip=src src-port=src_port dst-ip=dst dst-port=dst_port method=method path=path query=query version=req.ver headers=req.hdrs body=req.body
    event on-frontend-http-request

spoe-message coraza-res
    args app=str(sample_app) id=unique-id version=res.ver status=status headers=res.hdrs body=res.body
    event on-http-response
```

Instead of hard coded application name `str(sample_app)` you can use some HAProxy's variable. For example, frontend name `fe_name` or some custom variable.

The engine is in the scope `coraza`. So to enable it, you must set [the following line](https://github.com/corazawaf/coraza-spoa/blob/88b4e54ab3ddcb58d946ed1d6389eff73745575b/doc/config/haproxy.cfg#L21) in a `frontend` / `listener` section HAProxy config:
```haproxy
    filter spoe engine coraza config /etc/haproxy/coraza.cfg
    ...

```haproxy
# /etc/haproxy/haproxy.cfg
frontend web
    mode http
    bind :80
    unique-id-format %[uuid()]
    unique-id-header X-Unique-ID
    filter spoe engine coraza config /etc/haproxy/coraza.cfg

    log-format "%ci:%cp\ [%t]\ %ft\ %b/%s\ %Th/%Ti/%TR/%Tq/%Tw/%Tc/%Tr/%Tt\ %ST\ %B\ %CC\ %CS\ %tsc\ %ac/%fc/%bc/%sc/%rc\ %sq/%bq\ %hr\ %hs\ %{+Q}r\ %ID\ spoa-error:\%[var(txn.coraza.error)]\ waf-action:\%[var(txn.coraza.action)]"
    
    # Currently haproxy cannot use variables to set the code or deny_status, so this needs to be manually configured here
    http-request redirect code 302 location %[var(txn.coraza.data)] if { var(txn.coraza.action) -m str redirect }
    http-response redirect code 302 location %[var(txn.coraza.data)] if { var(txn.coraza.action) -m str redirect }

    http-request deny deny_status 403 hdr waf-block "request"  if { var(txn.coraza.action) -m str deny }
    http-response deny deny_status 403 hdr waf-block "response" if { var(txn.coraza.action) -m str deny }

    http-request silent-drop if { var(txn.coraza.action) -m str drop }
    http-response silent-drop if { var(txn.coraza.action) -m str drop }

    # Deny in case of an error, when processing with the Coraza SPOA
    http-request deny deny_status 504 if { var(txn.coraza.error) -m int gt 0 }
    http-response deny deny_status 504 if { var(txn.coraza.error) -m int gt 0 }

    # use web_backend to send filtered requests to the web application
    use_backend web_backend
```

Because, in the SPOE configuration file (coraza.cfg), we declare to use the backend [coraza-spoa](https://github.com/corazawaf/coraza-spoa/blob/88b4e54ab3ddcb58d946ed1d6389eff73745575b/doc/config/coraza.cfg#L14) to communicate with the service, so we need also to define it in the [HAProxy file](https://github.com/corazawaf/coraza-spoa/blob/dd5eb86d1e9abbdd5fe568249f36a6d85257eba7/doc/config/haproxy.cfg#L37):
```haproxy
backend coraza-spoa
    ...

```haproxy
# /etc/haproxy/haproxy.cfg
backend coraza-spoa
    mode tcp
    balance roundrobin
    timeout connect 5s # greater than hello timeout
    timeout server 3m  # greater than idle timeout
    server s1 127.0.0.1:9000
```

## Docker

- Build the coraza-spoa image `docker-compose build`
- Run haproxy, coraza-spoa and a mock server `docker-compose up`
- Perform a request which gets blocked by the WAF: `curl http://localhost:4000/\?x\=/etc/passwd`