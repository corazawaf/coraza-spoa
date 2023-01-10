// Copyright The OWASP Coraza contributors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"fmt"
	"net"
	"time"

	"github.com/corazawaf/coraza/v3/types"
	spoe "github.com/criteo/haproxy-spoe-go"
	"go.uber.org/zap"
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
		tx      types.Transaction
	)
	var app *application
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
				if len(s.defaultApplication) > 0 {
					app, ok = s.applications[s.defaultApplication]
					if !ok {
						return nil, fmt.Errorf("default application not found: %s", s.defaultApplication)
					}
					app.logger.Debug("application not found, using default", zap.Any("application", arg.Value), zap.String("default", s.defaultApplication))
				} else {
					return nil, fmt.Errorf("application not found: %v", arg.Value)
				}
			}
		case "id":
			id, ok := arg.Value.(string)
			if !ok {
				return nil, fmt.Errorf("invalid argument for http request id, string expected, got %v", arg.Value)
			}
			tx = app.waf.NewTransactionWithID(id)
			//tx.Variables.UniqueID.Set(tx.ID)
			defer func() {
				if tx.IsInterrupted() {
					tx.ProcessLogging()
					if err := tx.Close(); err != nil {
						app.logger.Error("failed to close transaction", zap.String("transaction_id", id), zap.String("error", err.Error()))
					}
				} else {
					err := app.cache.SetWithExpire(id, tx, time.Second*time.Duration(app.cfg.TransactionTTL))
					if err != nil {
						app.logger.Error(fmt.Sprintf("failed to cache transaction: %s", err.Error()))
					}
				}
			}()
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
				app.logger.Error(fmt.Sprintf("invalid argument for http request path, string expected, got %v", arg.Value))
				path = "/"
			}
		case "query":
			query, ok = arg.Value.(string)
			if !ok && arg.Value != nil {
				app.logger.Error(fmt.Sprintf("invalid argument for http request query, string expected, got %v", arg.Value))
				query = ""
			}
		case "version":
			version, ok = arg.Value.(string)
			if !ok {
				app.logger.Error(fmt.Sprintf("invalid argument for http request version, string expected, got %v", arg.Value))
				version = "1.1"
			}
		case "headers":
			value, ok := arg.Value.(string)
			if !ok {
				app.logger.Error(fmt.Sprintf("invalid argument for http request headers, string expected, got %v", arg.Value))
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
				return nil, fmt.Errorf("invalid argument for http request body, []byte expected, got %v", arg.Value)
			}

			_, err := tx.RequestBodyWriter().Write(body)
			if err != nil {
				return nil, err
			}
		default:
			app.logger.Error("invalid message on the http frontend request", zap.String("name", arg.Name), zap.Any("value", arg.Value))
		}
	}

	//app.logger.Debug(fmt.Sprintf("ProcessConnection: %s:%d -> %s:%d", srcIP.String(), srcPort, dstIP.String(), dstPort))
	tx.ProcessConnection(srcIP.String(), srcPort, dstIP.String(), dstPort)

	//app.logger.Debug(fmt.Sprintf("ProcessURI: %s %s?%s %s", method, path, query, "HTTP/"+version))
	tx.ProcessURI(path+"?"+query, method, "HTTP/"+version)

	if it := tx.ProcessRequestHeaders(); it != nil {
		return s.processInterruption(it, hit), nil
	}
	it, err := tx.ProcessRequestBody()
	if err != nil {
		return nil, err
	}
	if it != nil {
		return s.processInterruption(it, hit), nil
	}
	return s.message(miss), nil
}
