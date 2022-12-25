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

	"github.com/corazawaf/coraza/v3/types"
	spoe "github.com/criteo/haproxy-spoe-go"
	"go.uber.org/zap"
)

func (s *SPOA) processResponse(msg spoe.Message) ([]spoe.Action, error) {
	var (
		ok      bool
		app     *application
		id      = ""
		status  = 0
		version = ""
		tx      types.Transaction
	)
	defer func() {
		app.cache.Remove(id)
	}()

	for msg.Args.Next() {
		arg := msg.Args.Arg
		if arg.Name != "app" && app == nil {
			return nil, fmt.Errorf("app is not set")
		}
		switch arg.Name {
		case "app":
			var ok bool
			app, ok = s.applications[arg.Value.(string)]
			if !ok {
				return nil, fmt.Errorf("application %q not found", arg.Value.(string))
			}
		case "id":
			id, ok := arg.Value.(string)
			if !ok {
				return nil, fmt.Errorf("invalid argument for http response id, string expected, got %v", arg.Value)
			}

			txInterface, err := app.cache.Get(id)
			if err != nil {
				app.logger.Error("failed to get transaction from cache", zap.String("transaction_id", id), zap.String("error", err.Error()), zap.String("app", app.name))
				break
			}

			if tx, ok = txInterface.(types.Transaction); !ok {
				app.logger.Error("Application cache is corrupted", zap.String("transaction_id", id), zap.String("app", app.name))
				return nil, fmt.Errorf("application cache is corrupted")
			}
		case "version":
			version, ok = arg.Value.(string)
			if !ok {
				app.logger.Error(fmt.Sprintf("invalid argument for http response version, string expected, got %v", arg.Value))
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
				app.logger.Error(fmt.Sprintf("invalid argument for http response headers, string expected, got %v", arg.Value))
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
			_, err := tx.ResponseBodyWriter().Write(body)
			if err != nil {
				return nil, err
			}
		default:
			app.logger.Warn(fmt.Sprintf("invalid message on the http response, name: %s, value: %s", arg.Name, arg.Value))
		}
	}

	if it := tx.ProcessResponseHeaders(status, "HTTP/"+version); it != nil {
		return s.processInterruption(it, hit), nil
	}
	it, err := tx.ProcessResponseBody()
	if err != nil {
		return nil, err
	}
	if it != nil {
		return s.processInterruption(it, hit), nil
	}

	return s.message(miss), nil
}
