//go:build e2e

package internal

import (
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/corazawaf/coraza/v3/http/e2e"
	"github.com/mccutchen/go-httpbin/v2/httpbin"
	"github.com/rs/zerolog"

	"github.com/dropmorepackets/haproxy-go/pkg/testutil"
)

func TestE2E(t *testing.T) {
	t.Parallel()
	t.Run("coraza e2e suite", withCoraza(t, func(t *testing.T, config testutil.HAProxyConfig, bin string) {
		err := e2e.Run(e2e.Config{
			NulledBody:        false,
			ProxiedEntrypoint: "http://127.0.0.1:" + config.FrontendPort,
			HttpbinEntrypoint: bin,
		})
		if err != nil {
			t.Fatalf("e2e tests failed: %v", err)
		}
	}))
}

func withCoraza(t *testing.T, f func(*testing.T, testutil.HAProxyConfig, string)) func(t *testing.T) {
	s := httptest.NewServer(httpbin.New())
	t.Cleanup(s.Close)

	logger := zerolog.New(os.Stderr)

	appCfg := AppConfig{
		Directives:       e2e.Directives,
		ResponseCheck:    true,
		Logger:           logger,
		TransactionTTLMS: 10000,
	}

	application, err := appCfg.NewApplication()
	if err != nil {
		t.Fatal(err)
	}

	a := Agent{
		Context: context.Background(),
		Applications: map[string]*Application{
			"default": application,
		},
		Logger: logger,
	}

	// create the listener synchronously to prevent a race
	l := testutil.TCPListener(t)
	// ignore errors as the listener will be closed by t.Cleanup
	go a.Serve(l)

	cfg := testutil.HAProxyConfig{
		EngineAddr:   l.Addr().String(),
		FrontendPort: fmt.Sprintf("%d", testutil.TCPPort(t)),
		CustomFrontendConfig: `
    # Currently haproxy cannot use variables to set the code or deny_status, so this needs to be manually configured here
    http-request redirect code 302 location %[var(txn.e2e.data)] if { var(txn.e2e.action) -m str redirect }
    http-response redirect code 302 location %[var(txn.e2e.data)] if { var(txn.e2e.action) -m str redirect }

    acl is_deny var(txn.e2e.action) -m str deny
    acl status_424 var(txn.e2e.status) -m int 424

    # Special check for e2e tests as they validate the config.
    http-request deny deny_status 424 hdr waf-block "request"  if is_deny status_424
    http-response deny deny_status 424 hdr waf-block "response" if is_deny status_424

    http-request deny deny_status 403 hdr waf-block "request"  if is_deny
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
    args app=str(default) src-ip=src src-port=src_port dst-ip=dst dst-port=dst_port method=method path=path query=query version=req.ver headers=req.hdrs body=req.body
    event on-frontend-http-request

spoe-message coraza-res
    args app=str(default) id=var(txn.e2e.id) version=res.ver status=status headers=res.hdrs body=res.body
    event on-http-response
`,
		BackendConfig: fmt.Sprintf(`
mode http
server httpbin %s
`, s.Listener.Addr().String()),
	}

	return testutil.WithHAProxy(cfg, func(t *testing.T) {
		f(t, cfg, s.URL)
	})
}
