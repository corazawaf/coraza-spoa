// Copyright The OWASP Coraza contributors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"fmt"

	"github.com/corazawaf/coraza/v3/types"
	spoe "github.com/criteo/haproxy-spoe-go"
	"go.uber.org/zap"
)

type response struct {
	app     string
	id      string
	version string
	status  int
	headers string
	body    []byte
}

func NewResponse(msg message) (*response, error) {
	resp := response{}
	var err error

	resp.app, err = msg.App()
	if err != nil {
		return nil, err
	}

	resp.id, err = msg.Id()
	if err != nil {
		return nil, err
	}

	resp.version, err = msg.Version()
	if err != nil {
		fmt.Println(err.Error())
		resp.version = "1.1"
	}

	resp.status, err = msg.Status()
	if err != nil {
		return nil, err
	}

	resp.headers, err = msg.Headers()
	if err != nil {
		fmt.Println(err.Error())
	}

	resp.body, err = msg.Body()

	return &resp, nil
}

func (s *SPOA) processResponse(spoeMsg spoe.Message) ([]spoe.Action, error) {
	var (
		err  error
		resp *response
		app  *application
		tx   types.Transaction
	)
	defer func() {
		app.cache.Remove(resp.id)
	}()

	msg := NewMessage(spoeMsg)
	resp, err = NewResponse(msg)
	if err != nil {
		return nil, err
	}

	app, err = s.getApplication(resp.app)
	if err != nil {
		return nil, err
	}

	txInterface, err := app.cache.Get(resp.id)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction from cache", zap.String("transaction_id", resp.id), zap.String("error", err.Error()), zap.String("app", app.name))
	}
	tx, ok := txInterface.(types.Transaction)
	if !ok {
		return nil, fmt.Errorf("application cache is corrupted", zap.String("transaction_id", resp.id), zap.String("app", app.name))
	}

	headers, err := s.readHeaders(resp.headers)
	if err != nil {
		return nil, err
	}
	for key, values := range headers {
		for _, v := range values {
			tx.AddResponseHeader(key, v)
		}
	}

	it, _, err := tx.WriteResponseBody(resp.body)
	if err != nil {
		return nil, err
	}
	if it != nil {
		return s.processInterruption(it, hit), nil
	}

	it = tx.ProcessResponseHeaders(resp.status, "HTTP/"+resp.version)
	if it != nil {
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
