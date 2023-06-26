// Copyright The OWASP Coraza contributors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"fmt"
	"net"

	spoe "github.com/criteo/haproxy-spoe-go"
)

type message struct {
	msg  *spoe.Message
	args map[string]interface{}
}

func NewMessage(msg *spoe.Message) (*message, error) {
	message := message{
		msg:  msg,
		args: make(map[string]interface{}, msg.Args.Count()),
	}
	return &message, nil
}

func (m *message) findArg(name string) (interface{}, error) {
	argVal, exist := m.args[name]
	if exist {
		return argVal, nil
	}

	ai := m.msg.Args
	for ai.Next() {
		m.args[ai.Arg.Name] = ai.Arg.Value
		if ai.Arg.Name == name {
			return ai.Arg.Value, nil
		}
	}

	return nil, &ArgNotFoundError{name}
}

func (m *message) getStringArg(name string) (string, error) {
	argVal, err := m.findArg(name)
	if err != nil {
		return "", err
	}
	if argVal == nil {
		return "", nil
	}
	val, ok := argVal.(string)
	if !ok {
		return "", &typeMismatchError{name, "string", argVal}
	}
	return val, nil
}

func (m *message) getIntArg(name string) (int, error) {
	argVal, err := m.findArg(name)
	if err != nil {
		return 0, err
	}
	if argVal == nil {
		return 0, nil
	}
	val, ok := argVal.(int)
	if !ok {
		return 0, &typeMismatchError{name, "int", argVal}
	}
	return val, nil
}

func (m *message) getByteArrayArg(name string) ([]byte, error) {
	argVal, err := m.findArg(name)
	if err != nil {
		return nil, err
	}
	if argVal == nil {
		return nil, nil
	}
	val, ok := argVal.([]byte)
	if !ok {
		return nil, &typeMismatchError{name, "[]byte", argVal}
	}
	return val, nil
}

func (m *message) getIpArg(name string) (net.IP, error) {
	argVal, err := m.findArg(name)
	if err != nil {
		return nil, err
	}
	if argVal == nil {
		return nil, nil
	}
	val, ok := argVal.(net.IP)
	if !ok {
		return nil, &typeMismatchError{name, "net.IP", argVal}
	}
	return val, nil
}

type ArgNotFoundError struct {
	argName string
}

func (e *ArgNotFoundError) Error() string {
	return fmt.Sprintf("Argument '%s' not found", e.argName)
}

type typeMismatchError struct {
	key          string
	expectedType string
	actualValue  interface{}
}

func (e *typeMismatchError) Error() string {
	return fmt.Sprintf("Invalid argument for %s, %s expected, got %T", e.key, e.expectedType, e.actualValue)
}
