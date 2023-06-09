// Copyright The OWASP Coraza contributors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"fmt"

	"github.com/corazawaf/coraza/v3/types"
	spoe "github.com/criteo/haproxy-spoe-go"
	"go.uber.org/zap"
)

func (s *SPOA) processResponse(msg spoe.Message) ([]spoe.Action, error) {
	var (
		app     *application
		id      = ""
		status  = 0
		version = ""
		tx      types.Transaction
	)
	defer func() {
		app.cache.Remove(id)
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
	txInterface, err := app.cache.Get(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction from cache", zap.String("transaction_id", id), zap.String("error", err.Error()), zap.String("app", app.name))
	}
	tx, ok := txInterface.(types.Transaction)
	if !ok {
		return nil, fmt.Errorf("application cache is corrupted", zap.String("transaction_id", id), zap.String("app", app.name))
	}

	version, err = getVersion(args)
	if err != nil {
		app.logger.Error(err.Error())
	}

	status, err = getStatus(args)
	if err != nil {
		return nil, err
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
			tx.AddResponseHeader(key, v)
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

	if it := tx.ProcessResponseHeaders(status, "HTTP/"+version); it != nil {
		return s.processInterruption(it, hit), nil
	}

	it, err = tx.ProcessResponseBody()
	if err != nil {
		return nil, err
	}
	if it != nil {
		return s.processInterruption(it, hit), nil
	}

	return s.message(miss), nil
}
