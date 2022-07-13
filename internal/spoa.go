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
	"strings"

	"github.com/bluele/gcache"
	"github.com/corazawaf/coraza-spoa/config"
	"github.com/corazawaf/coraza-spoa/pkg/logger"
	"github.com/corazawaf/coraza/v2"
	"github.com/corazawaf/coraza/v2/seclang"
	spoe "github.com/criteo/haproxy-spoe-go"
	"go.uber.org/zap"

	_ "github.com/jptosso/coraza-libinjection"
	_ "github.com/jptosso/coraza-pcre"
)

const (
	// Miss sets the detection result to safe.
	Miss = iota
	// Hit opposite to Miss.
	Hit
)

// SPOA store the relevant data for starting SPOA.
type SPOA struct {
	cfg   *config.SPOA
	waf   *coraza.Waf
	cache gcache.Cache
}

// Start starts the SPOA to detect the security risks.
func (s *SPOA) Start() error {
	logger.Info("Starting SPOA")

	agent := spoe.New(func(messages *spoe.MessageIterator) ([]spoe.Action, error) {
		for messages.Next() {
			msg := messages.Message

			switch msg.Name {
			case "coraza-req":
				return s.processRequest(msg)
			case "coraza-res":
				return s.processResponse(msg)
			default:
				logger.Error(fmt.Sprintf("unsupported message: %s", msg.Name))
			}
		}
		return nil, nil
	})
	if err := agent.ListenAndServe(s.cfg.Bind); err != nil {
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
			return nil, fmt.Errorf("invalid header: %s", header)
		}

		h.Add(strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1]))
	}
	return h, nil
}

// New creates a new SPOA instance.
func New(cfg *config.SPOA) (*SPOA, error) {
	s := new(SPOA)
	s.cfg = cfg

	s.waf = coraza.NewWaf()
	parser, _ := seclang.NewParser(s.waf)
	if len(s.cfg.Include) == 0 {
		logger.Warn("No include path or file specified")
	}

	for _, f := range s.cfg.Include {
		if err := parser.FromFile(f); err != nil {
			return nil, err
		}
	}

	s.cache = gcache.New(s.cfg.TransactionActiveLimit).
		EvictedFunc(func(key, value interface{}) {
			// everytime a transaction is timedout we clean it
			tx, ok := value.(*coraza.Transaction)
			if !ok {
				return
			}
			// Process Logging won't do anything if TX was already logged.
			tx.ProcessLogging()
			if err := tx.Clean(); err != nil {
				logger.Error("Failed to clean cache", zap.Error(err))
			}
		}).ARC().Build()
	return s, nil
}
