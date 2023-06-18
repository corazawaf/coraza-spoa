// Copyright The OWASP Coraza contributors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"fmt"
	"net"

	spoe "github.com/criteo/haproxy-spoe-go"
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

func NewRequest(msg spoe.Message) (*request, error) {
	args := msg.Args.Map()
	req := request{}
	var err error

	req.app, err = getString(args, "app")
	if err != nil {
		return nil, err
	}

	req.id, err = getString(args, "id")
	if err != nil {
		return nil, err
	}

	req.srcIp, err = getIp(args, "src-ip")
	if err != nil {
		return nil, err
	}

	req.srcPort, err = getInt(args, "src-port")
	if err != nil {
		return nil, err
	}

	req.dstIp, err = getIp(args, "dst-ip")
	if err != nil {
		return nil, err
	}

	req.dstPort, err = getInt(args, "dst-port")
	if err != nil {
		return nil, err
	}

	req.method, err = getString(args, "method")
	if err != nil {
		return nil, err
	}

	req.path, err = getString(args, "path")
	if err != nil {
		fmt.Println(err.Error())
	}

	req.query, err = getString(args, "query")
	if err != nil {
		fmt.Println(err.Error())
	}

	req.version, err = getString(args, "version")
	if err != nil {
		fmt.Println(err.Error())
	}

	req.headers, err = getString(args, "headers")
	if err != nil {
		fmt.Println(err.Error())
	}

	req.body, _ = getByteArray(args, "body")

	return &req, nil
}
