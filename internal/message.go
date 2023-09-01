// Copyright The OWASP Coraza contributors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"fmt"
	"net/netip"

	"github.com/dropmorepackets/haproxy-go/pkg/encoding"
)

type message struct {
	args map[string]any
}

func NewMessage(m *encoding.Message) (*message, error) {
	msg := message{
		args: make(map[string]any),
	}

	e := encoding.AcquireKVEntry()
	defer encoding.ReleaseKVEntry(e)

	for m.KV.Next(e) {
		switch e.Type() {
		case encoding.DataTypeInt32, encoding.DataTypeInt64,
			encoding.DataTypeUInt32, encoding.DataTypeUInt64:
			msg.args[string(e.NameBytes())] = int(e.ValueInt())
		default:
			msg.args[string(e.NameBytes())] = e.Value()
		}
	}

	return &msg, nil
}

func (m *message) findArg(name string) (any, error) {
	argVal, exist := m.args[name]
	if exist {
		return argVal, nil
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

func (m *message) getIpArg(name string) (netip.Addr, error) {
	argVal, err := m.findArg(name)
	if err != nil {
		return netip.Addr{}, err
	}
	if argVal == nil {
		return netip.Addr{}, nil
	}
	val, ok := argVal.(netip.Addr)
	if !ok {
		return netip.Addr{}, &typeMismatchError{name, "netip.Addr", argVal}
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
	actualValue  any
}

func (e *typeMismatchError) Error() string {
	return fmt.Sprintf("Invalid argument for %s, %s expected, got %T", e.key, e.expectedType, e.actualValue)
}
