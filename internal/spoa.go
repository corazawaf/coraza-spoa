// Copyright The OWASP Coraza contributors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/bluele/gcache"
	"github.com/corazawaf/coraza-spoa/config"
	"github.com/corazawaf/coraza-spoa/log"
	"github.com/corazawaf/coraza/v3"
	"github.com/corazawaf/coraza/v3/types"
	spoe "github.com/criteo/haproxy-spoe-go"
)

const (
	// miss sets the detection result to safe.
	miss = iota
	// hit opposite to Miss.
	hit
)

type application struct {
	name  string
	cfg   *config.Application
	waf   coraza.WAF
	cache gcache.Cache
}

// SPOA store the relevant data for starting SPOA.
type SPOA struct {
	applications       map[string]*application
	defaultApplication string
}

// Start starts the SPOA to detect the security risks.
func (s *SPOA) Start(bind string) error {
	log.Debug().Msg("Starting SPOA")

	agent := spoe.New(func(messages *spoe.MessageIterator) ([]spoe.Action, error) {
		for messages.Next() {
			msg := messages.Message

			switch msg.Name {
			case "coraza-req":
				return s.processRequest(&msg)
			case "coraza-res":
				return s.processResponse(&msg)
			}
		}
		return nil, nil
	})
	if err := agent.ListenAndServe(bind); err != nil {
		return err
	}
	return nil
}

func (s *SPOA) processInterruption(it *types.Interruption, code int) []spoe.Action {
	//if it.Status == 0 {
	//  tx.variables.responseStatus.Set("", []string{"403"})
	//} else {
	//  status := strconv.Itoa(int(it.Status))
	//  tx.variables.responseStatus.Set("", []string{status})
	//}

	return []spoe.Action{
		spoe.ActionSetVar{
			Name:  "status",
			Scope: spoe.VarScopeTransaction,
			Value: it.Status,
		},
		spoe.ActionSetVar{
			Name:  "action",
			Scope: spoe.VarScopeTransaction,
			Value: it.Action,
		},
		spoe.ActionSetVar{
			Name:  "data",
			Scope: spoe.VarScopeTransaction,
			Value: it.Data,
		},
		spoe.ActionSetVar{
			Name:  "ruleid",
			Scope: spoe.VarScopeTransaction,
			Value: it.RuleID,
		},
	}
}

func (s *SPOA) message(code int) []spoe.Action {
	return []spoe.Action{
		spoe.ActionSetVar{
			Name:  "fail",
			Scope: spoe.VarScopeTransaction,
			Value: code,
		},
	}
}

func (s *SPOA) error(code int, err error) []spoe.Action {
	return []spoe.Action{
		spoe.ActionSetVar{
			Name:  "err_code",
			Scope: spoe.VarScopeTransaction,
			Value: code,
		},
		spoe.ActionSetVar{
			Name:  "err_msg",
			Scope: spoe.VarScopeTransaction,
			Value: err.Error(),
		},
	}
}

func (s *SPOA) badRequestError(err error) []spoe.Action {
	log.Error().Err(err).Msg("Bad request")
	return s.error(1, err)
}

func (s *SPOA) badResponseError(err error) []spoe.Action {
	log.Error().Err(err).Msg("Bad response")
	return s.error(2, err)
}

func (s *SPOA) processRequestError(err error) []spoe.Action {
	return s.error(3, err)
}

func (s *SPOA) processResponseError(err error) []spoe.Action {
	return s.error(4, err)
}

func (s *SPOA) readHeaders(headers string) (http.Header, error) {
	h := http.Header{}
	hs := strings.Split(headers, "\r\n")

	for _, header := range hs {
		if header == "" {
			continue
		}

		kv := strings.SplitN(header, ":", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid header: %q", header)
		}

		h.Add(strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1]))
	}
	return h, nil
}

// New Create a new SPOA instance.
func New(conf *config.Config) (*SPOA, error) {
	apps := make(map[string]*application)
	for name, cfg := range conf.Applications {
		wafConf := coraza.NewWAFConfig().
			WithDirectives(cfg.Directives)

		if conf.Log.Waf {
			wafConf = wafConf.WithErrorCallback(log.WafErrorCallback)
		}

		//nolint:staticcheck // https://github.com/golangci/golangci-lint/issues/741
		if len(cfg.Rules) > 0 {
			// Deprecated: this will soon be removed
			log.Warn().Msg("'rules' directive in configuration is deprecated and will be removed soon, use 'directives' instead")
			wafConf = wafConf.WithDirectives(strings.Join(cfg.Rules, "\n"))
		}

		waf, err := coraza.NewWAF(wafConf)
		if err != nil {
			log.Error().Err(err).Str("app", name).Msg("Unable to create WAF instance")
			return nil, err
		}

		app := &application{
			name: name,
			cfg:  cfg,
			waf:  waf,
		}

		app.cache = gcache.New(app.cfg.TransactionActiveLimit).
			EvictedFunc(func(key, value interface{}) {
				// everytime a transaction is timedout we clean it
				tx, ok := value.(types.Transaction)
				if !ok {
					return
				}
				// Process Logging won't do anything if TX was already logged.
				tx.ProcessLogging()
				if err := tx.Close(); err != nil {
					tx.DebugLogger().Error().Err(err).Msg("Failed to clean cache")
				}
			}).LFU().Expiration(time.Millisecond * time.Duration(cfg.TransactionTTLMilliseconds)).Build()

		apps[name] = app
	}
	return &SPOA{
		applications:       apps,
		defaultApplication: conf.DefaultApplication,
	}, nil
}

func (s *SPOA) getApplication(appName string) (*application, error) {
	var app *application

	// Looking for app by name from message
	if appName != "" {
		app, exist := s.applications[appName]
		if exist {
			return app, nil
		}
	}

	// Looking for app by default app name
	app, exist := s.applications[s.defaultApplication]
	if exist {
		log.Debug().Str("application", appName).Str("default app", s.defaultApplication).Msg("Application not found, using default")
		return app, nil
	}

	return nil, fmt.Errorf("application not found, application %s, default: %s", appName, s.defaultApplication)
}

func (s *SPOA) processRequest(spoeMsg *spoe.Message) ([]spoe.Action, error) {
	var (
		err error
		req *request
		app *application
		tx  types.Transaction
	)

	defer func() {
		if tx == nil || app == nil {
			return
		}
		if tx.IsInterrupted() {
			tx.ProcessLogging()
			if err := tx.Close(); err != nil {
				tx.DebugLogger().Error().Err(err).Str("transaction_id", tx.ID()).Msg("Failed to close transaction")
			}
		} else {
			if app.cfg.NoResponseCheck {
				return
			}
			err := app.cache.SetWithExpire(tx.ID(), tx, time.Millisecond*time.Duration(app.cfg.TransactionTTLMilliseconds))
			if err != nil {
				log.Error().Err(err).Msg("failed to cache transaction")
			}
		}
	}()

	req, err = NewRequest(spoeMsg)
	if err != nil {
		return s.badRequestError(err), nil
	}

	app, err = s.getApplication(req.app)
	if err != nil {
		return s.badRequestError(err), nil
	}

	tx = app.waf.NewTransactionWithID(req.id)
	if tx.IsRuleEngineOff() {
		log.Warn().Msg("Rule engine is Off, Coraza is not going to process any rule")
		return s.message(miss), nil
	}

	err = req.init()
	if err != nil {
		return s.badRequestError(err), nil
	}

	headers, err := s.readHeaders(req.headers)
	if err != nil {
		return s.badRequestError(err), nil
	}
	for key, values := range headers {
		for _, v := range values {
			tx.AddRequestHeader(key, v)
		}
	}

	it, _, err := tx.WriteRequestBody(req.body)
	if err != nil {
		tx.DebugLogger().Error().Err(err).Str("transaction_id", tx.ID()).Msg("Failed to write request body")
		return s.processRequestError(err), nil
	}
	if it != nil {
		return s.processInterruption(it, hit), nil
	}

	tx.ProcessConnection(string(req.srcIp), req.srcPort, string(req.dstIp), req.dstPort)
	tx.ProcessURI(req.path+"?"+req.query, req.method, "HTTP/"+req.version)

	it = tx.ProcessRequestHeaders()
	if it != nil {
		return s.processInterruption(it, hit), nil
	}

	it, err = tx.ProcessRequestBody()
	if err != nil {
		tx.DebugLogger().Error().Err(err).Str("transaction_id", tx.ID()).Msg("Failed to process request body")
		return s.processRequestError(err), nil
	}
	if it != nil {
		return s.processInterruption(it, hit), nil
	}

	return s.message(miss), nil
}

func (s *SPOA) processResponse(spoeMsg *spoe.Message) ([]spoe.Action, error) {
	var (
		err  error
		resp *response
		app  *application
		tx   types.Transaction
	)
	defer func() {
		app.cache.Remove(resp.id)
	}()

	resp, err = NewResponse(spoeMsg)
	if err != nil {
		return s.badResponseError(err), nil
	}

	app, err = s.getApplication(resp.app)
	if err != nil {
		return s.badResponseError(err), nil
	}

	txInterface, err := app.cache.Get(resp.id)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction from cache, transaction_id: %s, app: %s, error: %s", resp.id, app.name, err.Error())
	}
	tx, ok := txInterface.(types.Transaction)
	if !ok {
		return nil, fmt.Errorf("application cache is corrupted, transaction_id: %s, app: %s", resp.id, app.name)
	}

	err = resp.init()
	if err != nil {
		return s.badResponseError(err), nil
	}

	headers, err := s.readHeaders(resp.headers)
	if err != nil {
		return s.badResponseError(err), nil
	}
	for key, values := range headers {
		for _, v := range values {
			tx.AddResponseHeader(key, v)
		}
	}

	it, _, err := tx.WriteResponseBody(resp.body)
	if err != nil {
		tx.DebugLogger().Error().Err(err).Str("transaction_id", tx.ID()).Msg("Failed to write response body")
		return s.processResponseError(err), nil
	}
	if it != nil {
		return s.processInterruption(it, hit), nil
	}

	it = tx.ProcessResponseHeaders(resp.status, "HTTP/"+resp.version)
	if it != nil {
		return s.processInterruption(it, hit), nil
	}

	it, err = tx.ProcessResponseBody()
	if err != nil {
		tx.DebugLogger().Error().Err(err).Str("transaction_id", tx.ID()).Msg("Failed to process response body")
		return s.processResponseError(err), nil
	}
	if it != nil {
		return s.processInterruption(it, hit), nil
	}

	return s.message(miss), nil
}
