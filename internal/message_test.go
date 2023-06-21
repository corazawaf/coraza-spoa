// Copyright The OWASP Coraza contributors
// SPDX-License-Identifier: Apache-2.0

//go:build !tinygo
// +build !tinygo

package internal

import (
	"net"
	"reflect"
	"testing"
)

type testData struct {
	key           string
	value         interface{}
	requestKey    string
	expectedValue interface{}
	expectedError reflect.Type
}

var (
	keyNotFoundErrorType  = reflect.TypeOf((*keyNotFoundError)(nil))
	typeMismatchErrorType = reflect.TypeOf((*typeMismatchError)(nil))
)

func TestGetString(t *testing.T) {
	testMatrix := []testData{
		{"non-empty", "some-value", "non-empty", "some-value", nil},
		{"empty", "", "empty", "", nil},
		{"null", nil, "null", "", nil},
		{"int-type", 1, "int-type", "", typeMismatchErrorType},
		{"map-key", "value", "other-key", "", keyNotFoundErrorType},
	}

	for _, data := range testMatrix {
		args := map[string]interface{}{
			data.key: data.value,
		}
		output, err := getString(args, data.requestKey)
		if output != data.expectedValue {
			t.Errorf("getString(%v, %v): output '%v' not equal to expected '%v'", args, data.key, output, data.expectedValue)
		}
		errType := reflect.TypeOf(err)
		if data.expectedError != errType {
			t.Errorf("getString(%v, %v): error '%v' not equal to expected '%v'", args, data.key, errType, data.expectedError)
		}
	}
}

func TestGetInt(t *testing.T) {
	testMatrix := []testData{
		{"non-zero", 11, "non-zero", 11, nil},
		{"zero", 0, "zero", 0, nil},
		{"null", nil, "null", 0, nil},
		{"string-type", "1", "string-type", 0, typeMismatchErrorType},
		{"map-key", 11, "other-key", 0, keyNotFoundErrorType},
	}

	for _, data := range testMatrix {
		args := map[string]interface{}{
			data.key: data.value,
		}
		output, err := getInt(args, data.requestKey)
		if output != data.expectedValue {
			t.Errorf("getInt(%v, %v): output '%v' not equal to expected '%v'", args, data.key, output, data.expectedValue)
		}
		errType := reflect.TypeOf(err)
		if data.expectedError != errType {
			t.Errorf("getString(%v, %v): error '%v' not equal to expected '%v'", args, data.key, errType, data.expectedError)
		}
	}
}

func TestGetByteArray(t *testing.T) {
	testMatrix := []testData{
		{"non-empty", []byte{99, 100}, "non-empty", []byte{99, 100}, nil},
		{"empty", []byte{}, "empty", []byte{}, nil},
		{"null", nil, "null", nil, nil},
		{"string-type", "str", "string-type", nil, typeMismatchErrorType},
		{"map-key", []byte{}, "other-key", nil, keyNotFoundErrorType},
	}

	for _, data := range testMatrix {
		args := map[string]interface{}{
			data.key: data.value,
		}
		output, err := getByteArray(args, data.requestKey)
		if !(output == nil && data.expectedValue == nil || reflect.DeepEqual(output, data.expectedValue)) {
			t.Errorf("getByteArray(%v, %v): output '%v' not equal to expected '%v'", args, data.key, output, data.expectedValue)
		}
		errType := reflect.TypeOf(err)
		if data.expectedError != errType {
			t.Errorf("getString(%v, %v): error '%v' not equal to expected '%v'", args, data.key, errType, data.expectedError)
		}
	}
}

func TestGetIp(t *testing.T) {
	testMatrix := []testData{
		{"ipv4", net.IPv4(192, 168, 1, 1), "ipv4", net.IPv4(192, 168, 1, 1), nil},
		{"ipv6", net.ParseIP("::1"), "ipv6", net.ParseIP("::1"), nil},
		{"invalid-ipv4", net.ParseIP("321.456.789.10"), "invalid-ipv4", net.ParseIP("321.456.789.10"), nil},
		{"null", nil, "null", nil, nil},
		{"invalid-type", "127.0.0.1", "invalid-type", nil, typeMismatchErrorType},
		{"map-key", net.IPv4(192, 168, 1, 1), "other-key", nil, keyNotFoundErrorType},
	}

	for _, data := range testMatrix {
		args := map[string]interface{}{
			data.key: data.value,
		}
		output, err := getIp(args, data.requestKey)
		if !(output == nil && data.expectedValue == nil || data.expectedValue.(net.IP).Equal(output)) {
			t.Errorf("getIp(%v, %v): output '%v' not equal to expected '%v'", args, data.key, output, data.expectedValue)
		}
		errType := reflect.TypeOf(err)
		if data.expectedError != errType {
			t.Errorf("getString(%v, %v): error '%v' not equal to expected '%v'", args, data.key, errType, data.expectedError)
		}
	}
}
