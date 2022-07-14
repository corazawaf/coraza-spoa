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
	"net"
	"time"

	"github.com/corazawaf/coraza/v2"
	"github.com/corazawaf/coraza/v2/types/variables"
	spoe "github.com/criteo/haproxy-spoe-go"

	"github.com/corazawaf/coraza-spoa/pkg/logger"
)

func (s *SPOA) processRequest(msg spoe.Message) ([]spoe.Action, error) {
	var (
		ok      bool
		method  = ""
		path    = "/"
		query   = ""
		version = "1.1"
		srcIP   net.IP
		srcPort = 0
		dstIP   net.IP
		dstPort = 0
		tx      = new(coraza.Transaction)
	)

	for msg.Args.Next() {
		arg := msg.Args.Arg

		switch arg.Name {
		case "id":
			tx = s.waf.NewTransaction()
			tx.ID, ok = arg.Value.(string)
			if !ok {
				return nil, fmt.Errorf("invalid argument for http request id, string expected, got %v", arg.Value)
			}
			tx.GetCollection(variables.UniqueID).Set("", []string{tx.ID})
		case "src-ip":
			srcIP, ok = arg.Value.(net.IP)
			if !ok {
				return nil, fmt.Errorf("invalid argument for src ip, net.IP expected, got %v", arg.Value)
			}
		case "src-port":
			srcPort, ok = arg.Value.(int)
			if !ok {
				return nil, fmt.Errorf("invalid argument for src port, integer expected, got %v", arg.Value)
			}
		case "dst-ip":
			dstIP, ok = arg.Value.(net.IP)
			if !ok {
				return nil, fmt.Errorf("invalid argument for dst ip, net.IP expected, got %v", arg.Value)
			}
		case "dst-port":
			dstPort, ok = arg.Value.(int)
			if !ok {
				return nil, fmt.Errorf("invalid argument for dst port, integer expected, got %v", arg.Value)
			}
		case "method":
			method, ok = arg.Value.(string)
			if !ok {
				return nil, fmt.Errorf("invalid argument for http request method, string expected, got %v", arg.Value)
			}
		case "path":
			path, ok = arg.Value.(string)
			if !ok {
				logger.Error(fmt.Sprintf("invalid argument for http request path, string expected, got %v", arg.Value))
				path = "/"
			}
		case "query":
			query, ok = arg.Value.(string)
			if !ok && arg.Value != nil {
				logger.Error(fmt.Sprintf("invalid argument for http request query, string expected, got %v", arg.Value))
				query = ""
			}
		case "version":
			version, ok = arg.Value.(string)
			if !ok {
				logger.Error(fmt.Sprintf("invalid argument for http request version, string expected, got %v", arg.Value))
				version = "1.1"
			}
		case "headers":
			value, ok := arg.Value.(string)
			if !ok {
				logger.Error(fmt.Sprintf("invalid argument for http request headers, string expected, got %v", arg.Value))
				value = ""
			}

			headers, err := s.readHeaders(value)
			if err != nil {
				return nil, err
			}

			for key, values := range headers {
				for _, v := range values {
					tx.AddRequestHeader(key, v)
				}
			}
		case "body":
			body, ok := arg.Value.([]byte)
			if !ok {
				return nil, fmt.Errorf("invalid argument for http reqeust body, []byte expected, got %v", arg.Value)
			}

			_, err := tx.RequestBodyBuffer.Write(body)
			if err != nil {
				return nil, err
			}
		default:
			logger.Warn(fmt.Sprintf("invalid message on the http frontend request, name: %s, value: %s", arg.Name, arg.Value))
		}
	}

	defer func() {
		if tx.Interrupted() {
			//tx.AuditEngine = types.AuditEngineOn
			tx.ProcessLogging()
			if err := tx.Clean(); err != nil {
				logger.Error("failed to clean transaction", logger.String("transaction_id", tx.ID), logger.String("error", err.Error()))
			}
		} else {
			err := s.cache.SetWithExpire(tx.ID, tx, time.Millisecond*time.Duration(s.cfg.TransactionTTL))
			if err != nil {
				logger.Error(fmt.Sprintf("failed to cache transaction: %s", err.Error()))
			}
		}
	}()

	logger.Debug(fmt.Sprintf("ProcessConnection: %s:%d -> %s:%d", srcIP.String(), srcPort, dstIP.String(), dstPort))
	tx.ProcessConnection(srcIP.String(), srcPort, dstIP.String(), dstPort)

	logger.Debug(fmt.Sprintf("ProcessURI: %s?%s %s %s", path, query, method, "HTTP/"+version))
	tx.ProcessURI(path+"?"+query, method, "HTTP/"+version)

	if it := tx.ProcessRequestHeaders(); it != nil {
		return s.message(Hit), nil
	}
	it, err := tx.ProcessRequestBody()
	if err != nil {
		return nil, err
	}
	if it != nil {
		logger.Info(fmt.Sprintf("Request hit occured due to rule ID %d from source IP %s", it.RuleID, srcIP.String()))
		return s.message(Hit), nil
	}
	return s.message(Miss), nil
}
