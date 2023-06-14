// Copyright The OWASP Coraza contributors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"fmt"
	"net"
)

type request struct {
	app     string
	id      string
	srcIp   net.IP
	srcPort int
	dstIp   net.IP
	dstPort int
	method  string
	path    string
	query   string
	version string
	headers string
	body    []byte
}

func NewRequest(msg message) (*request, error) {
	req := request{}
	var err error

	req.app, err = msg.App()
	if err != nil {
		return nil, err
	}

	req.id, err = msg.Id()
	if err != nil {
		return nil, err
	}

	req.srcIp, err = msg.SrcIp()
	if err != nil {
		return nil, err
	}

	req.srcPort, err = msg.SrcPort()
	if err != nil {
		return nil, err
	}

	req.dstIp, err = msg.DstIp()
	if err != nil {
		return nil, err
	}

	req.dstPort, err = msg.DstPort()
	if err != nil {
		return nil, err
	}

	req.method, err = msg.Method()
	if err != nil {
		return nil, err
	}

	req.path, err = msg.Path()
	if err != nil {
		fmt.Println(err.Error())
		req.path = "/"
	}

	req.query, err = msg.Query()
	if err != nil {
		fmt.Println(err.Error())
	}

	req.version, err = msg.Version()
	if err != nil {
		fmt.Println(err.Error())
		req.version = "1.1"
	}

	req.headers, err = msg.Headers()
	if err != nil {
		fmt.Println(err.Error())
	}

	req.body, err = msg.Body()

	return &req, nil
}
