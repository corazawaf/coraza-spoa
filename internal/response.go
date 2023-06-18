// Copyright The OWASP Coraza contributors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"fmt"

	spoe "github.com/criteo/haproxy-spoe-go"
)

type response struct {
	app     string
	id      string
	version string
	status  int
	headers string
	body    []byte
}

func NewResponse(msg spoe.Message) (*response, error) {
	args := msg.Args.Map()
	resp := response{}
	var err error

	resp.app, err = getString(args, "app")
	if err != nil {
		return nil, err
	}

	resp.id, err = getString(args, "id")
	if err != nil {
		return nil, err
	}

	resp.version, err = getString(args, "version")
	if err != nil {
		fmt.Println(err.Error())
	}

	resp.status, err = getInt(args, "status")
	if err != nil {
		return nil, err
	}

	resp.headers, err = getString(args, "headers")
	if err != nil {
		fmt.Println(err.Error())
	}

	resp.body, _ = getByteArray(args, "body")

	return &resp, nil
}
