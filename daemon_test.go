package main

import (
	"net"
	"os"
	"strings"
	"testing"
)

func TestSdNotify(t *testing.T) {
	socketAddr := &net.UnixAddr{
		Name: "/tmp/coraza-spoa-daemon.sock",
		Net:  "unixgram",
	}
	socket, err := net.ListenUnixgram(socketAddr.Net, socketAddr)
	if err != nil {
		t.Fatal(err)
	}

	defer os.Remove(socketAddr.Name)

	t.Setenv("NOTIFY_SOCKET", socketAddr.Name)

	tests := []struct {
		name     string
		messages []string
	}{
		{
			"Ready",
			[]string{SdNotifyReady},
		},
		{
			"Reloading",
			[]string{SdNotifyReloading},
		},
		{
			"Stopping",
			[]string{SdNotifyStopping},
		},
		{
			"Ready with status",
			[]string{SdNotifyReady, SdNotifyStatus + "Test Ready"},
		},
		{
			"Reloading with status",
			[]string{SdNotifyReloading, SdNotifyStatus + "Test Reloading"},
		},
		{
			"Stopping with status",
			[]string{SdNotifyStopping, SdNotifyStatus + "Test Stopping"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := SdNotify(tt.messages...)
			if err != nil {
				t.Errorf("SdNotify() error = %v", err)
			}

			// getting the messages
			var buf [1024]byte
			n, err := socket.Read(buf[:])
			if err != nil {
				t.Fatal(err)
			}

			var received = string(buf[:n])
			var expected = strings.Join(tt.messages, "\n")
			if received != expected {
				t.Errorf("SdNotify() = returned:\n---\n%v\n---\nWanted:\n---\n%v\n---", received, expected)
			}
		})
	}
}

func TestSdNotify_NoSocketSet(t *testing.T) {
	t.Setenv("NOTIFY_SOCKET", "")

	err := SdNotify(SdNotifyReady)
	if err != nil {
		t.Errorf("SdNotify() error = not nil, %v", err)
	}
}

func TestSdNotify_WrongSocketSet(t *testing.T) {
	t.Setenv("NOTIFY_SOCKET", "/tmp/coraza-spoa-wrong.sock")

	err := SdNotify(SdNotifyReady)
	if err != nil && err.Error() == "dial unixgram /tmp/coraza-spoa-wrong.sock: connect: no such file or directory" {
		return
	}
	t.Error("SdNotify() no error")
}
