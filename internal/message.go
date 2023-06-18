// Copyright The OWASP Coraza contributors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"fmt"
	"net"
)

func getString(args map[string]interface{}, key string) (string, error) {
	argVal, exist := args[key]
	if exist && argVal != nil {
		val, ok := argVal.(string)
		if ok {
			return val, nil
		}
	}

	return "", fmt.Errorf("invalid argument for %s, string expected, got %v", key, argVal)
}

func getInt(args map[string]interface{}, key string) (int, error) {
	argVal, exist := args[key]
	if exist && argVal != nil {
		val, ok := argVal.(int)
		if ok {
			return val, nil
		}
	}

	return 0, fmt.Errorf("invalid argument for %s, integer expected, got %v", key, argVal)
}

func getByteArray(args map[string]interface{}, key string) ([]byte, error) {
	argVal, exist := args[key]
	if exist && argVal != nil {
		val, ok := argVal.([]byte)
		if ok {
			return val, nil
		}
	}

	return nil, fmt.Errorf("invalid argument for %s, []byte expected, got %v", key, argVal)
}

func getIp(args map[string]interface{}, key string) (net.IP, error) {
	argVal, exist := args[key]
	if exist && argVal != nil {
		val, ok := argVal.(net.IP)
		if ok {
			return val, nil
		}
	}

	return nil, fmt.Errorf("invalid argument for %s, net.IP expected, got %v", key, argVal)
}
