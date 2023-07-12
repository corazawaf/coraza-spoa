// Copyright The OWASP Coraza contributors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"net"

	"github.com/corazawaf/coraza-spoa/log"
	spoe "github.com/criteo/haproxy-spoe-go"
)

type request struct {
	msg     *message
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

func NewRequest(spoeMsg *spoe.Message) (*request, error) {
	msg, err := NewMessage(spoeMsg)
	if err != nil {
		return nil, err
	}

	request := request{}
	request.msg = msg

	request.app, err = msg.getStringArg("app")
	if err != nil {
		return nil, err
	}

	request.id, err = request.msg.getStringArg("id")
	if err != nil {
		return nil, err
	}

	return &request, nil
}

func (req *request) init() error {
	var err error

	req.srcIp, err = req.msg.getIpArg("src-ip")
	if err != nil {
		return err
	}

	req.srcPort, err = req.msg.getIntArg("src-port")
	if err != nil {
		return err
	}

	req.dstIp, err = req.msg.getIpArg("dst-ip")
	if err != nil {
		return err
	}

	req.dstPort, err = req.msg.getIntArg("dst-port")
	if err != nil {
		return err
	}

	req.method, err = req.msg.getStringArg("method")
	if err != nil {
		return err
	}

	req.path, err = req.msg.getStringArg("path")
	if err != nil {
		log.Trace().Err(err).Msg("Can't get Path from HTTP Request")
	}

	req.query, err = req.msg.getStringArg("query")
	if err != nil {
		log.Trace().Err(err).Msg("Can't get Query from HTTP Request")
	}

	req.version, err = req.msg.getStringArg("version")
	if err != nil {
		log.Trace().Err(err).Msg("Can't get Version from HTTP Request")
	}

	req.headers, err = req.msg.getStringArg("headers")
	if err != nil {
		log.Trace().Err(err).Msg("Can't get Headers from HTTP Request")
	}

	req.body, _ = req.msg.getByteArrayArg("body")

	return nil
}
