// Copyright 2022 The Corazawaf Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
	"github.com/corazawaf/coraza/v3/seclang"
	"github.com/corazawaf/coraza/v3/types"
	spoe "github.com/criteo/haproxy-spoe-go"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	// miss sets the detection result to safe.
	miss = iota
	// hit opposite to Miss.
	hit
)

type application struct {
	name   string
	cfg    *config.Application
	waf    *coraza.Waf
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

		app := &application{
			name:   name,
			cfg:    cfg,
			waf:    coraza.NewWaf(),
			logger: zap.New(core),
		}
		app.waf.SetErrorLogCb(func(err types.MatchedRule) {
			app.logger.Error(err.ErrorLog(500))
			switch err.Rule.Severity {
			case types.RuleSeverityCritical:
			case types.RuleSeverityEmergency:
			case types.RuleSeverityError:
			case types.RuleSeverityWarning:
			case types.RuleSeverityNotice:
			case types.RuleSeverityInfo:
			case types.RuleSeverityDebug:

			}
		})
		parser, _ := seclang.NewParser(app.waf)
		for _, f := range app.cfg.Include {
			if err := parser.FromFile(f); err != nil {
				return nil, err
			}
		}

		app.cache = gcache.New(app.cfg.TransactionActiveLimit).
			EvictedFunc(func(key, value interface{}) {
				// everytime a transaction is timedout we clean it
				tx, ok := value.(*coraza.Transaction)
				if !ok {
					return
				}
				// Process Logging won't do anything if TX was already logged.
				tx.ProcessLogging()
				if err := tx.Clean(); err != nil {
					app.logger.Error("Failed to clean cache", zap.Error(err))
				}
			}).LFU().Expiration(time.Duration(cfg.TransactionTTL) * time.Second).Build()
	}
	return &SPOA{
		applications: apps,
	}, nil
}
