// Copyright The OWASP Coraza contributors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"fmt"
	"net"

	spoe "github.com/criteo/haproxy-spoe-go"
)

type message struct {
	args map[string]interface{}
}

func NewMessage(spmsg spoe.Message) message {
	return message{
		args: spmsg.Args.Map(),
	}
}

func (msg message) App() (string, error) {
	appArgVal, exist := msg.args["app"]
	if exist && appArgVal != nil {
		id, ok := appArgVal.(string)
		if ok {
			return id, nil
		}
	}

	return "", fmt.Errorf("invalid argument for application name, string expected, got %v", appArgVal)
}

func (msg message) Id() (string, error) {
	idArgVal, exist := msg.args["id"]
	if exist && idArgVal != nil {
		id, ok := idArgVal.(string)
		if ok {
			return id, nil
		}
	}

	return "", fmt.Errorf("invalid argument for id, string expected, got %v", idArgVal)
}

func (msg message) SrcIp() (net.IP, error) {
	srcIpArgVal, exist := msg.args["src-ip"]
	if exist && srcIpArgVal != nil {
		srcIp, ok := srcIpArgVal.(net.IP)
		if ok {
			return srcIp, nil
		}
	}

	return nil, fmt.Errorf("invalid argument for src ip, net.IP expected, got %v", srcIpArgVal)
}

func (msg message) SrcPort() (int, error) {
	srcPortArgVal, exist := msg.args["src-port"]
	if exist && srcPortArgVal != nil {
		srcPort, ok := srcPortArgVal.(int)
		if ok {
			return srcPort, nil
		}
	}

	return 0, fmt.Errorf("invalid argument for src port, integer expected, got %v", srcPortArgVal)
}

func (msg message) DstIp() (net.IP, error) {
	dstIpArgVal, exist := msg.args["dst-ip"]
	if exist && dstIpArgVal != nil {
		dstIp, ok := dstIpArgVal.(net.IP)
		if ok {
			return dstIp, nil
		}
	}

	return nil, fmt.Errorf("invalid argument for dst ip, net.IP expected, got %v", dstIpArgVal)
}

func (msg message) DstPort() (int, error) {
	dstPortArgVal, exist := msg.args["dst-port"]
	if exist && dstPortArgVal != nil {
		dstPort, ok := dstPortArgVal.(int)
		if ok {
			return dstPort, nil
		}
	}

	return 0, fmt.Errorf("invalid argument for dst port, integer expected, got %v", dstPortArgVal)
}

func (msg message) Method() (string, error) {
	methodArgVal, exist := msg.args["method"]
	if exist && methodArgVal != nil {
		method, ok := methodArgVal.(string)
		if ok {
			return method, nil
		}
	}

	return "", fmt.Errorf("invalid argument for http method, string expected, got %v", methodArgVal)
}

func (msg message) Path() (string, error) {
	pathArgVal, exist := msg.args["path"]
	if exist && pathArgVal != nil {
		path, ok := pathArgVal.(string)
		if ok {
			return path, nil
		}
	}

	return "/", fmt.Errorf("invalid argument for http path, string expected, got %v", pathArgVal)
}

func (msg message) Query() (string, error) {
	queryArgVal, exist := msg.args["query"]
	if exist && queryArgVal != nil {
		query, ok := queryArgVal.(string)
		if ok {
			return query, nil
		}
	}

	return "", fmt.Errorf("invalid argument for http query, string expected, got %v", queryArgVal)
}

func (msg message) Version() (string, error) {
	versionArgVal, exist := msg.args["version"]
	if exist && versionArgVal != nil {
		version, ok := versionArgVal.(string)
		if ok {
			return version, nil
		}
	}

	return "1.1", fmt.Errorf("invalid argument for http version, string expected, got %v", versionArgVal)
}

func (msg message) Headers() (string, error) {
	headersArgVal, exist := msg.args["headers"]
	if exist && headersArgVal != nil {
		headers, ok := headersArgVal.(string)
		if ok {
			return headers, nil
		}
	}

	return "", fmt.Errorf("invalid argument for http headers, string expected, got %v", headersArgVal)
}

func (msg message) Body() ([]byte, error) {
	bodyArgVal, exist := msg.args["body"]
	if exist && bodyArgVal != nil {
		body, ok := bodyArgVal.([]byte)
		if ok {
			return body, nil
		}
	}

	return nil, fmt.Errorf("invalid argument for http body, []byte expected, got %v", bodyArgVal)
}

func (msg message) Status() (int, error) {
	statusArgVal, exist := msg.args["status"]
	if exist && statusArgVal != nil {
		dstPort, ok := statusArgVal.(int)
		if ok {
			return dstPort, nil
		}
	}

	return 0, fmt.Errorf("invalid argument for http response status, int expected, got %v", statusArgVal)
}
