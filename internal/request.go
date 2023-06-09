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
		app     *application
		id      = ""
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

	args := msg.Args.Map()
	var err error

	appName, _ := getAppName(args)
	app, err = s.getApplication(appName)
	if err != nil {
		return nil, err
	}

	id, err = getId(args)
	if err != nil {
		return nil, err
	}
	tx = app.waf.NewTransactionWithID(id)

	srcIP, _ = getSourceIp(args)
	if err != nil {
		return nil, err
	}

	srcPort, _ = getSourcePort(args)
	if err != nil {
		return nil, err
	}

	dstIP, _ = getDestinationIp(args)
	if err != nil {
		return nil, err
	}

	dstPort, _ = getDestinationPort(args)
	if err != nil {
		return nil, err
	}

	method, _ = getMethod(args)
	if err != nil {
		return nil, err
	}

	path, err = getPath(args)
	if err != nil {
		app.logger.Error(err.Error())
	}

	query, _ = getQuery(args)
	if err != nil {
		app.logger.Error(err.Error())
	}

	version, err = getVersion(args)
	if err != nil {
		app.logger.Error(err.Error())
	}

	headersString, err := getHeaders(args)
	if err != nil {
		app.logger.Error(err.Error())
	}
	headers, err := s.readHeaders(headersString)
	if err != nil {
		return nil, err
	}
	for key, values := range headers {
		for _, v := range values {
			tx.AddRequestHeader(key, v)
		}
	}

	body, _ := getBody(args)
	it, _, err := tx.WriteRequestBody(body)
	if err != nil {
		return nil, err
	}
	if it != nil {
		return s.processInterruption(it, hit), nil
	}

	app.logger.Debug(fmt.Sprintf("ProcessConnection: %s:%d -> %s:%d", srcIP.String(), srcPort, dstIP.String(), dstPort))
	tx.ProcessConnection(srcIP.String(), srcPort, dstIP.String(), dstPort)

	app.logger.Debug(fmt.Sprintf("ProcessURI: %s %s?%s %s", method, path, query, "HTTP/"+version))
	tx.ProcessURI(path+"?"+query, method, "HTTP/"+version)

	if it := tx.ProcessRequestHeaders(); it != nil {
		return s.processInterruption(it, hit), nil
	}

	it, err = tx.ProcessRequestBody()
	if err != nil {
		return nil, err
	}
	if it != nil {
		return s.processInterruption(it, hit), nil
	}

	return s.message(miss), nil
}
