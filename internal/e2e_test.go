//go:build e2e

package internal

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/corazawaf/coraza/v3/http/e2e"
	"github.com/dropmorepackets/haproxy-go/pkg/testutil"
	"github.com/mccutchen/go-httpbin/v2/httpbin"
	"github.com/rs/zerolog"
)

func TestE2E(t *testing.T) {
	t.Run("coraza e2e suite", func(t *testing.T) {
		config, bin, _ := runCoraza(t)
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
		config, _, _ := runCoraza(t)

		if os.Getenv("CI") != "" {
			t.Skip("CI is too slow for this test.")
		}

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
	})
}

func runCoraza(tb testing.TB) (testutil.HAProxyConfig, string, string) {
	s := httptest.NewServer(httpbin.New())
	tb.Cleanup(s.Close)

	logger := zerolog.New(os.Stderr).With().Timestamp().Logger()

	appCfg := AppConfig{
		Directives:     e2e.Directives,
		ResponseCheck:  true,
		Logger:         logger,
		TransactionTTL: 10 * time.Second,
	}

	application, err := appCfg.NewApplication()
	if err != nil {
		tb.Fatal(err)
	}

	a := Agent{
		Context:            context.Background(),
		DefaultApplication: application,
		Applications: map[string]*Application{
			"default": application,
		},
		Logger: logger,
	}

	// create the listener synchronously to prevent a race
	l := testutil.TCPListener(tb)
	// ignore errors as the listener will be closed by t.Cleanup
	go a.Serve(l)

	cfg := testutil.HAProxyConfig{
		EngineAddr:   l.Addr().String(),
		FrontendPort: fmt.Sprintf("%d", testutil.TCPPort(tb)),
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

	frontendSocket := cfg.Run(tb)

	return cfg, s.URL, frontendSocket
}
