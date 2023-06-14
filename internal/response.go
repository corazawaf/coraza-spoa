// Copyright The OWASP Coraza contributors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"fmt"
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

	resp.body, _ = msg.Body()

	return &resp, nil
}
