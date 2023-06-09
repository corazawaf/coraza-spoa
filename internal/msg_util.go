// Copyright The OWASP Coraza contributors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"fmt"
	"net"
)

func getAppName(args map[string]interface{}) (string, error) {
	appArgVal, exist := args["app"]
	if exist && appArgVal != nil {
		id, ok := appArgVal.(string)
		if ok {
			return id, nil
		}
	}

	return "", fmt.Errorf("invalid argument for application name, string expected, got %v", appArgVal)
}

func getId(args map[string]interface{}) (string, error) {
	idArgVal, exist := args["id"]
	if exist && idArgVal != nil {
		id, ok := idArgVal.(string)
		if ok {
			return id, nil
		}
	}

	return "", fmt.Errorf("invalid argument for id, string expected, got %v", idArgVal)
}

func getSourceIp(args map[string]interface{}) (net.IP, error) {
	srcIpArgVal, exist := args["src-ip"]
	if exist && srcIpArgVal != nil {
		srcIp, ok := srcIpArgVal.(net.IP)
		if ok {
			return srcIp, nil
		}
	}

	return nil, fmt.Errorf("invalid argument for src ip, net.IP expected, got %v", srcIpArgVal)
}

func getSourcePort(args map[string]interface{}) (int, error) {
	srcPortArgVal, exist := args["src-port"]
	if exist && srcPortArgVal != nil {
		srcPort, ok := srcPortArgVal.(int)
		if ok {
			return srcPort, nil
		}
	}

	return 0, fmt.Errorf("invalid argument for src port, integer expected, got %v", srcPortArgVal)
}

func getDestinationIp(args map[string]interface{}) (net.IP, error) {
	dstIpArgVal, exist := args["dst-ip"]
	if exist && dstIpArgVal != nil {
		dstIp, ok := dstIpArgVal.(net.IP)
		if ok {
			return dstIp, nil
		}
	}

	return nil, fmt.Errorf("invalid argument for dst ip, net.IP expected, got %v", dstIpArgVal)
}

func getDestinationPort(args map[string]interface{}) (int, error) {
	dstPortArgVal, exist := args["dst-port"]
	if exist && dstPortArgVal != nil {
		dstPort, ok := dstPortArgVal.(int)
		if ok {
			return dstPort, nil
		}
	}

	return 0, fmt.Errorf("invalid argument for dst port, integer expected, got %v", dstPortArgVal)
}

func getMethod(args map[string]interface{}) (string, error) {
	methodArgVal, exist := args["method"]
	if exist && methodArgVal != nil {
		method, ok := methodArgVal.(string)
		if ok {
			return method, nil
		}
	}

	return "", fmt.Errorf("invalid argument for http method, string expected, got %v", methodArgVal)
}

func getPath(args map[string]interface{}) (string, error) {
	pathArgVal, exist := args["path"]
	if exist && pathArgVal != nil {
		path, ok := pathArgVal.(string)
		if ok {
			return path, nil
		}
	}

	return "/", fmt.Errorf("invalid argument for http path, string expected, got %v", pathArgVal)
}

func getQuery(args map[string]interface{}) (string, error) {
	queryArgVal, exist := args["query"]
	if exist && queryArgVal != nil {
		query, ok := queryArgVal.(string)
		if ok {
			return query, nil
		}
	}

	return "", fmt.Errorf("invalid argument for http query, string expected, got %v", queryArgVal)
}

func getVersion(args map[string]interface{}) (string, error) {
	versionArgVal, exist := args["version"]
	if exist && versionArgVal != nil {
		version, ok := versionArgVal.(string)
		if ok {
			return version, nil
		}
	}

	return "1.1", fmt.Errorf("invalid argument for http version, string expected, got %v", versionArgVal)
}

func getHeaders(args map[string]interface{}) (string, error) {
	headersArgVal, exist := args["headers"]
	if exist && headersArgVal != nil {
		headers, ok := headersArgVal.(string)
		if ok {
			return headers, nil
		}
	}

	return "", fmt.Errorf("invalid argument for http headers, string expected, got %v", headersArgVal)
}

func getBody(args map[string]interface{}) ([]byte, error) {
	bodyArgVal, exist := args["body"]
	if exist && bodyArgVal != nil {
		body, ok := bodyArgVal.([]byte)
		if ok {
			return body, nil
		}
	}

	return nil, fmt.Errorf("invalid argument for http body, []byte expected, got %v", bodyArgVal)
}

func getStatus(args map[string]interface{}) (int, error) {
	statusArgVal, exist := args["status"]
	if exist && statusArgVal != nil {
		dstPort, ok := statusArgVal.(int)
		if ok {
			return dstPort, nil
		}
	}

	return 0, fmt.Errorf("invalid argument for http response status, int expected, got %v", statusArgVal)
}
