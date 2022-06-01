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

	"github.com/corazawaf/coraza-spoa/pkg/logger"
	"github.com/corazawaf/coraza/v2"
	"github.com/corazawaf/coraza/v2/types/variables"
	spoe "github.com/criteo/haproxy-spoe-go"
)

func (s *SPOA) resetTX(id string) *coraza.Transaction {
	tx := s.waf.NewTransaction()
	tx.ID = id
	tx.GetCollection(variables.UniqueID).Set("", []string{tx.ID})
	return tx
}

func (s *SPOA) processResponse(msg spoe.Message) ([]spoe.Action, error) {
	var (
		ok      bool
		id      = ""
		status  = 0
		version = ""
		tx      = new(coraza.Transaction)
	)
	defer func() {
		// This will also force the transaction to be closed
		s.cache.Remove(id)
	}()

	for msg.Args.Next() {
		arg := msg.Args.Arg

		switch arg.Name {
		case "id":
			id, ok := arg.Value.(string)
			if !ok {
				return nil, fmt.Errorf("invalid argument for http response id, string expected, got %v", arg.Value)
			}

			txInterface, err := s.cache.Get(id)
			if err != nil {
				logger.Error("failed to get transaction from cache", logger.String("transaction_id", id), logger.String("error", err.Error()))
				tx = s.resetTX(id)
				break
			}

			if tx, ok = txInterface.(*coraza.Transaction); ok {
				break
			}
			tx = s.resetTX(id)
		case "version":
			version, ok = arg.Value.(string)
			if !ok {
				logger.Error(fmt.Sprintf("invalid argument for http response version, string expected, got %v", arg.Value))
				version = "1.1"
			}
		case "status":
			status, ok = arg.Value.(int)
			if !ok {
				return nil, fmt.Errorf("invalid argument for http response status, int expected, got %v", arg.Value)
			}
		case "headers":
			value, ok := arg.Value.(string)
			if !ok {
				logger.Error(fmt.Sprintf("invalid argument for http response headers, string expected, got %v", arg.Value))
				value = ""
			}
			headers, err := s.readHeaders(value)
			if err != nil {
				return nil, err
			}
			for key, values := range headers {
				for _, v := range values {
					tx.AddResponseHeader(key, v)
				}
			}
		case "body":
			body, ok := arg.Value.([]byte)
			if !ok {
				return nil, fmt.Errorf("invalid argument for http response body, []byte expected, got %v", arg.Value)
			}
			_, err := tx.ResponseBodyBuffer.Write(body)
			if err != nil {
				return nil, err
			}
		default:
			logger.Warn(fmt.Sprintf("invalid message on the http response, name: %s, value: %s", arg.Name, arg.Value))
		}
	}

	if it := tx.ProcessResponseHeaders(status, "HTTP/"+version); it != nil {
		return s.message(Hit), nil
	}
	it, err := tx.ProcessResponseBody()
	if err != nil {
		return nil, err
	}
	if it != nil {
		return s.message(Hit), nil
	}

	return s.message(Miss), nil
}
