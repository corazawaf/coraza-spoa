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
	"github.com/corazawaf/coraza/v3"
	"github.com/corazawaf/coraza/v3/types"
	spoe "github.com/criteo/haproxy-spoe-go"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/corazawaf/coraza-spoa/config"
)

const (
	// miss sets the detection result to safe.
	miss = iota
	// hit opposite to Miss.
	hit
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
	applications map[string]*application
}

// Start starts the SPOA to detect the security risks.
func (s *SPOA) Start(bind string) error {
	// s.logger.Info("Starting SPOA")

	agent := spoe.New(func(messages *spoe.MessageIterator) ([]spoe.Action, error) {
		for messages.Next() {
			msg := messages.Message

			switch msg.Name {
			case "coraza-req":
				return s.processRequest(msg)
			case "coraza-res":
				return s.processResponse(msg)
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
		// TODO - deprected, don't use this anymore.
		//  will be removed in a future version.
		spoe.ActionSetVar{
			Name:  "fail",
			Scope: spoe.VarScopeTransaction,
			Value: code,
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
		data := mr.ErrorLog(0)
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

// New creates a new SPOA instance.
func New(conf map[string]*config.Application) (*SPOA, error) {
	apps := make(map[string]*application)
	for name, cfg := range conf {
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
			WithDirectives(strings.Join(cfg.Rules, "\n")).
			WithErrorLogger(logError(logger))

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
			}).LFU().Expiration(time.Duration(cfg.TransactionTTL) * time.Second).Build()

		apps[name] = app
	}
	return &SPOA{
		applications: apps,
	}, nil
}
