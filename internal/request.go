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
	"net"
	"time"
)

func (s *SPOA) processRequest(msg spoe.Message) ([]spoe.Action, error) {
	var (
		method   = ""
		path     = ""
		query    = ""
		phase    = 0
		tx       = new(coraza.Transaction)
		argNames = []string{"Transaction ID", "Request IP", "Method", "Path", "Query", "HTTP Version",
			"Request Headers", "Request Body"}
	)

	for msg.Args.Next() {
		var (
			ok    bool
			value = ""
			arg   = msg.Args.Arg
		)

		if phase != 1 && phase != 7 {
			value, ok = arg.Value.(string)
			if !ok {
				return nil, fmt.Errorf("invalid argument for %s, string expected, got %v", argNames[phase], arg.Value)
			}
		}

		switch phase {
		case 0:
			tx = s.waf.NewTransaction()
			tx.ID = value
			tx.GetCollection(variables.UniqueID).Set("", []string{tx.ID})
		case 1:
			if val, ok := arg.Value.(net.IP); !ok {
				tx.ProcessConnection(val.String(), 0, "", 0)
			} else {
				return nil, fmt.Errorf("invalid argument for %s, net.IP expected, got %v", argNames[phase], arg.Value)
			}
		case 2:
			method = value
		case 3:
			path = value
		case 4:
			query = value
		case 5:
			tx.ProcessURI(path+"?"+query, method, "HTTP/"+value)
		case 6:
			headers, err := s.readHeaders(value)
			if err != nil {
				return nil, err
			}

			for key, values := range headers {
				for _, v := range values {
					tx.AddRequestHeader(key, v)
				}
			}
			if it := tx.ProcessRequestHeaders(); it != nil {
				return s.message(Hit), nil
			}
		case 7:
			body, ok := arg.Value.([]byte)
			if !ok {
				return nil, fmt.Errorf("invalid argument for %s, []byte expected, got %v", argNames[phase], arg.Value)
			}

			_, err := tx.RequestBodyBuffer.Write(body)
			if err != nil {
				return nil, err
			}
			it, err := tx.ProcessRequestBody()
			if err != nil {
				return nil, err
			}
			if it != nil {
				return s.message(Hit), nil
			}
		default:
			return nil, fmt.Errorf("invalid message on the http frontend request, name: %s, value: %s", arg.Name, arg.Value)
		}

		phase++
	}

	err := s.cache.SetWithExpire(tx.ID, tx, time.Millisecond*time.Duration(s.cfg.TransactionTTL))
	if err != nil {
		logger.Error(fmt.Sprintf("failed to cache transaction: %s", err.Error()))
	}
	return s.message(Miss), nil
}
