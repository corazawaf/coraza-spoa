// Copyright The OWASP Coraza contributors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"fmt"
	"net"
)

type keyNotFoundError struct {
	key string
}

func (e *keyNotFoundError) Error() string {
	return fmt.Sprintf("Key '%s' not found", e.key)
}

type typeMismatchError struct {
	key          string
	expectedType string
	actualValue  interface{}
}

func (e *typeMismatchError) Error() string {
	return fmt.Sprintf("Invalid argument for %s, %s expected, got %T", e.key, e.expectedType, e.actualValue)
}

func getString(args map[string]interface{}, key string) (string, error) {
	argVal, exist := args[key]
	if !exist {
		return "", &keyNotFoundError{key}
	}
	if argVal == nil {
		return "", nil
	}
	val, ok := argVal.(string)
	if !ok {
		return "", &typeMismatchError{key, "string", argVal}
	}
	return val, nil
}

func getInt(args map[string]interface{}, key string) (int, error) {
	argVal, exist := args[key]
	if !exist {
		return 0, &keyNotFoundError{key}
	}
	if argVal == nil {
		return 0, nil
	}
	val, ok := argVal.(int)
	if !ok {
		return 0, &typeMismatchError{key, "int", argVal}
	}
	return val, nil
}

func getByteArray(args map[string]interface{}, key string) ([]byte, error) {
	argVal, exist := args[key]
	if !exist {
		return nil, &keyNotFoundError{key}
	}
	if argVal == nil {
		return nil, nil
	}
	val, ok := argVal.([]byte)
	if !ok {
		return nil, &typeMismatchError{key, "[]byte", argVal}
	}
	return val, nil
}

func getIp(args map[string]interface{}, key string) (net.IP, error) {
	argVal, exist := args[key]
	if !exist {
		return nil, &keyNotFoundError{key}
	}
	if argVal == nil {
		return nil, nil
	}
	val, ok := argVal.(net.IP)
	if !ok {
		return nil, &typeMismatchError{key, "net.IP", argVal}
	}
	return val, nil
}
