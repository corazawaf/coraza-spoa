// Copyright The OWASP Coraza contributors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/bluele/gcache"
	"github.com/corazawaf/coraza-spoa/config"
	"github.com/corazawaf/coraza/v3"
	"github.com/corazawaf/coraza/v3/types"
	spoe "github.com/criteo/haproxy-spoe-go"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// TODO - in coraza v3 ErrorLogCallback is currently in the internal package
type ErrorLogCallback = func(rule types.MatchedRule)

type application struct {
	name   string
	cfg    *config.Application
	waf    coraza.WAF
	cache  gcache.Cache
	logger *zap.Logger
}

// SPOA store the relevant data for starting SPOA.
type SPOA struct {
	applications       map[string]*application
	defaultApplication string
}

// Start starts the SPOA to detect the security risks.
func (s *SPOA) Start(bind string) error {
	// s.logger.Info("Starting SPOA")

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
	defer s.cleanApplications()
	if err := agent.ListenAndServe(bind); err != nil {
		return err
	}
	return nil
}

func (s *SPOA) processInterruption(it *types.Interruption) []spoe.Action {
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

func (s *SPOA) allowAction() []spoe.Action {
	act := []spoe.Action{
		spoe.ActionSetVar{
			Name:  "action",
			Scope: spoe.VarScopeTransaction,
			Value: "allow",
		},
	}
	return act
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

func (s *SPOA) cleanApplications() {
	for _, app := range s.applications {
		if err := app.logger.Sync(); err != nil {
			app.logger.Error("failed to sync logger", zap.Error(err))
		}
	}
}

func logError(logger *zap.Logger) ErrorLogCallback {
	return func(mr types.MatchedRule) {
		data := mr.ErrorLog()
		switch mr.Rule().Severity() {
		case types.RuleSeverityEmergency:
			logger.Error(data)
		case types.RuleSeverityAlert:
			logger.Error(data)
		case types.RuleSeverityCritical:
			logger.Error(data)
		case types.RuleSeverityError:
			logger.Error(data)
		case types.RuleSeverityWarning:
			logger.Warn(data)
		case types.RuleSeverityNotice:
			logger.Info(data)
		case types.RuleSeverityInfo:
			logger.Info(data)
		case types.RuleSeverityDebug:
			logger.Debug(data)
		}
	}
}

// New Create a new SPOA instance.
func New(conf *config.Config) (*SPOA, error) {
	apps := make(map[string]*application)
	for name, cfg := range conf.Applications {
		pe := zap.NewProductionEncoderConfig()

		fileEncoder := zapcore.NewJSONEncoder(pe)

		pe.EncodeTime = zapcore.ISO8601TimeEncoder

		level, err := zapcore.ParseLevel(cfg.LogLevel)
		if err != nil {
			level = zap.InfoLevel
		}
		f, err := os.OpenFile(cfg.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, err
		}
		core := zapcore.NewTee(
			zapcore.NewCore(fileEncoder, zapcore.AddSync(f), level),
		)

		logger := zap.New(core)

		conf := coraza.NewWAFConfig().
			WithDirectives(cfg.Directives).
			WithErrorCallback(logError(logger))

		//nolint:staticcheck // https://github.com/golangci/golangci-lint/issues/741
		if len(cfg.Rules) > 0 {
			// Deprecated: this will soon be removed
			logger.Warn("'rules' directive in configuration is deprecated and will be removed soon, use 'directives' instead")
			conf = conf.WithDirectives(strings.Join(cfg.Rules, "\n"))
		}

		waf, err := coraza.NewWAF(conf)
		if err != nil {
			logger.Error("unable to create waf instance", zap.String("app", name), zap.Error(err))
			return nil, err
		}

		app := &application{
			name:   name,
			cfg:    cfg,
			waf:    waf,
			logger: logger,
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
					app.logger.Error("Failed to clean cache", zap.Error(err))
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
		app.logger.Debug("application not found, using default", zap.Any("application", appName), zap.String("default", s.defaultApplication))
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
				app.logger.Error("failed to close transaction", zap.String("transaction_id", tx.ID()), zap.String("error", err.Error()))
			}
		} else {
			if app.cfg.NoResponseCheck {
				return
			}
			err := app.cache.SetWithExpire(tx.ID(), tx, time.Millisecond*time.Duration(app.cfg.TransactionTTLMilliseconds))
			if err != nil {
				app.logger.Error(fmt.Sprintf("failed to cache transaction: %s", err.Error()))
			}
		}
	}()

	req, err = NewRequest(spoeMsg)
	if err != nil {
		return nil, err
	}

	app, err = s.getApplication(req.app)
	if err != nil {
		return nil, err
	}

	tx = app.waf.NewTransactionWithID(req.id)
	if tx.IsRuleEngineOff() {
		app.logger.Warn("Rule engine is Off, Coraza is not going to process any rule")
		return s.allowAction(), nil
	}

	err = req.init()
	if err != nil {
		return nil, err
	}

	headers, err := s.readHeaders(req.headers)
	if err != nil {
		return nil, err
	}
	for key, values := range headers {
		for _, v := range values {
			tx.AddRequestHeader(key, v)
		}
	}

	it, _, err := tx.WriteRequestBody(req.body)
	if err != nil {
		return nil, err
	}
	if it != nil {
		return s.processInterruption(it), nil
	}

	tx.ProcessConnection(string(req.srcIp), req.srcPort, string(req.dstIp), req.dstPort)
	tx.ProcessURI(req.path+"?"+req.query, req.method, "HTTP/"+req.version)

	it = tx.ProcessRequestHeaders()
	if it != nil {
		return s.processInterruption(it), nil
	}

	it, err = tx.ProcessRequestBody()
	if err != nil {
		return nil, err
	}
	if it != nil {
		return s.processInterruption(it), nil
	}

	return s.allowAction(), nil
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
		return nil, err
	}

	app, err = s.getApplication(resp.app)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	headers, err := s.readHeaders(resp.headers)
	if err != nil {
		return nil, err
	}
	for key, values := range headers {
		for _, v := range values {
			tx.AddResponseHeader(key, v)
		}
	}

	it, _, err := tx.WriteResponseBody(resp.body)
	if err != nil {
		return nil, err
	}
	if it != nil {
		return s.processInterruption(it), nil
	}

	it = tx.ProcessResponseHeaders(resp.status, "HTTP/"+resp.version)
	if it != nil {
		return s.processInterruption(it), nil
	}

	it, err = tx.ProcessResponseBody()
	if err != nil {
		return nil, err
	}
	if it != nil {
		return s.processInterruption(it), nil
	}

	return s.allowAction(), nil
}
