# Owasp Coraza Haproxy
[![Code Linting](https://github.com/corazawaf/coraza-spoa/actions/workflows/lint.yaml/badge.svg)](https://github.com/corazawaf/coraza-spoa/actions/workflows/lint.yaml)
[![CodeQL Scanning](https://github.com/corazawaf/coraza-spoa/actions/workflows/codeql.yaml/badge.svg)](https://github.com/corazawaf/coraza-spoa/actions/workflows/codeql.yaml)

## Overview
This is a third-party daemon that connects to SPOE. It sends the request and response sent by HAProxy to [OWASP Coraza](https://github.com/corazawaf/coraza) and returns the verdict.

## Compilation
### Build
The command `make` will compile the source code and produce the executable file `coraza-spoa`.

### Clean
When you need to re-compile the source code, you can use the command `make clean` to clean the executable file.

## Configuration file
The example configuration file is `config.yml.default`, you can copy it and modify the related configuration information.

## Start the service
After you have compiled it, you can start the service by running the command `./coraza-spoa`.
```shell
$> ./coraza-spoa -h
Usage of ./coraza-spoa:
  -config-file string
        The configuration file of the coraza-spoa. (default "./config.yml")
```

## Configure a SPOE to use the service
Here is the configuration template to use for your SPOE with OWASP Coraza module, you can find it in the [doc/config/coraza.cfg](https://github.com/corazawaf/coraza-spoa/blob/main/doc/config/coraza.cfg):
```editorconfig
[coraza]
spoe-agent coraza-agent
    messages coraza-req coraza-res
    option var-prefix coraza
    timeout hello      100ms
    timeout idle       2m
    timeout processing 10ms
    use-backend coraza-spoa
    log global

spoe-message coraza-req
    args id=unique-id src-ip=src method=method path=path query=query version=req.ver headers=req.hdrs bodyreq.body
    event on-frontend-http-request

spoe-message coraza-res
    args id=unique-id version=res.ver status=status headers=res.hdrs body=res.body
    event on-http-response
```

The engine is in the scope "coraza". So to enable it, you must set the following line in a frontend/listener section:
``` editorconfig
frontend coraza.io
    ...
    unique-id-format %[uuid()]
    unique-id-header X-Unique-ID
    filter spoe engine coraza config coraza.cfg
    ...
```

Because, in SPOE configuration file, we declare to use the backend "coraza-spoa" to communicate with the service, so we need to define it in the HAProxy file. For example:
```editorconfig
backend coraza-spoa
    mode tcp
    balance roundrobin
    timeout connect 5000ms
    timeout client 5000ms
    timeout server 5000ms
    server s1 127.0.0.1:9000
```

The OWASP Coraza action is returned in a variable named "txn.coraza.fail". It contains the verdict of the request. If the variable is set to 1, the request will be denied.
```editorconfig
http-request deny if { var(txn.coraza.fail) -m int eq 1 }
http-response deny if { var(txn.coraza.fail) -m int eq 1 }
```
With this rule, all unsafe requests will be rejected. You can find the example HAProxy configuration file in the [doc/config/haproxy.cfg](https://github.com/corazawaf/coraza-spoa/blob/main/doc/config/haproxy.cfg).



## Docker

- Build the coraza-spoa image `docker-compose build`
- Run haproxy, coraza-spoa and a mock server `docker-compose up`
- Perform a request which gets blocked by the WAF: `curl http://localhost:4000/\?x\=/etc/passwd`
