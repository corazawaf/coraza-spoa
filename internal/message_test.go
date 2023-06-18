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
	expectedValue interface{}
	expectedError string
}

func TestGetString(t *testing.T) {
	testMatrix := []testData{
		{"non-empty", "qweasd", "qweasd", ""},
		{"empty", "", "", ""},
		{"null", nil, "", "invalid argument for null, string expected, got <nil>"},
	}

	for _, data := range testMatrix {
		args := map[string]interface{}{
			data.key: data.value,
		}
		output, err := getString(args, data.key)
		if output != data.expectedValue {
			t.Errorf("getString(%v, %v): output '%v' not equal to expected '%v'", args, data.key, output, data.expectedValue)
		}
		if errorToString(err) != data.expectedError {
			t.Errorf("getString(%v, %v): error '%v' not equal to expected '%v'", args, data.key, err, data.expectedError)
		}
	}
}

func TestGetInt(t *testing.T) {
	testMatrix := []testData{
		{"non-zero", 11, 11, ""},
		{"zero", 0, 0, ""},
		{"null", nil, 0, "invalid argument for null, integer expected, got <nil>"},
	}

	for _, data := range testMatrix {
		args := map[string]interface{}{
			data.key: data.value,
		}
		output, err := getInt(args, data.key)
		if output != data.expectedValue {
			t.Errorf("getInt(%v, %v): output '%v' not equal to expected '%v'", args, data.key, output, data.expectedValue)
		}
		if errorToString(err) != data.expectedError {
			t.Errorf("getInt(%v, %v): error '%v' not equal to expected '%v'", args, data.key, err, data.expectedError)
		}
	}
}

func TestGetByteArray(t *testing.T) {
	testMatrix := []testData{
		{"non-empty", []byte{99, 100}, []byte{99, 100}, ""},
		{"empty", []byte{}, []byte{}, ""},
		{"null", nil, nil, "invalid argument for null, []byte expected, got <nil>"},
	}

	for _, data := range testMatrix {
		args := map[string]interface{}{
			data.key: data.value,
		}
		output, err := getByteArray(args, data.key)
		if !(output == nil && data.expectedValue == nil || reflect.DeepEqual(output, data.expectedValue)) {
			t.Errorf("getByteArray(%v, %v): output '%v' not equal to expected '%v'", args, data.key, output, data.expectedValue)
		}
		if errorToString(err) != data.expectedError {
			t.Errorf("getByteArray(%v, %v): error '%v' not equal to expected '%v'", args, data.key, err, data.expectedError)
		}
	}
}

func TestGetIp(t *testing.T) {
	testMatrix := []testData{
		{"ipv4", net.IPv4(192, 168, 1, 1), net.IPv4(192, 168, 1, 1), ""},
		{"ipv6", net.ParseIP("::1"), net.ParseIP("::1"), ""},
		{"invalid-ipv4", net.ParseIP("321.456.789.10"), net.ParseIP("321.456.789.10"), ""},
		{"invalid-type", "127.0.0.1", nil, "invalid argument for invalid-type, net.IP expected, got 127.0.0.1"},
		{"null", nil, nil, "invalid argument for null, net.IP expected, got <nil>"},
	}

	for _, data := range testMatrix {
		args := map[string]interface{}{
			data.key: data.value,
		}
		output, err := getIp(args, data.key)
		if !(output == nil && data.expectedValue == nil || data.expectedValue.(net.IP).Equal(output)) {
			t.Errorf("getIp(%v, %v): output '%v' not equal to expected '%v'", args, data.key, output, data.expectedValue)
		}
		if errorToString(err) != data.expectedError {
			t.Errorf("getIp(%v, %v): error '%v' not equal to expected '%v'", args, data.key, err, data.expectedError)
		}
	}
}

func errorToString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
