// Copyright The OWASP Coraza contributors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"fmt"

	"github.com/dropmorepackets/haproxy-go/pkg/encoding"
)

type response struct {
	msg     *message
	app     string
	id      string
	version string
	status  int
	headers string
	body    []byte
}

func NewResponse(spoeMsg *encoding.Message) (*response, error) {
	msg, err := NewMessage(spoeMsg)
	if err != nil {
		return nil, err
	}

	response := response{}
	response.msg = msg

	response.app, err = msg.getStringArg("app")
	if err != nil {
		return nil, err
	}

	response.id, err = response.msg.getStringArg("id")
	if err != nil {
		return nil, err
	}

	return &response, nil
}

func (resp *response) init() error {
	var err error

	resp.version, err = resp.msg.getStringArg("version")
	if err != nil {
		fmt.Println(err.Error())
	}

	resp.status, err = resp.msg.getIntArg("status")
	if err != nil {
		return err
	}

	resp.headers, err = resp.msg.getStringArg("headers")
	if err != nil {
		fmt.Println(err.Error())
	}

	resp.body, _ = resp.msg.getByteArrayArg("body")

	return nil
}
