// This provides the functionality for notifying the service manager about status changes via sd_notify
// see https://www.freedesktop.org/software/systemd/man/latest/sd_notify.html
package main

import (
	"net"
	"os"
	"strings"
)

const (
	// SdNotifyReady Tells the service manager that service startup is finished, or the service finished re-loading its configuration.
	SdNotifyReady = "READY=1"

	// SdNotifyReloading Tells the service manager that the service is beginning to reload its configuration.
	SdNotifyReloading = "RELOADING=1"

	// SdNotifyStopping Tells the service manager that the service is beginning its shutdown.
	SdNotifyStopping = "STOPPING=1"

	// SdNotifyStatus Passes a single-line UTF-8 status string back to the service manager that describes the service state.
	SdNotifyStatus = "STATUS="
)

// SdNotify Communicates with the NOTIFY_SOCKET
// Accepts
// Returns nil, if the socket doesn't exist, or the message was sent successfully
// Returns an error, if the message wasn't sent successfully
func SdNotify(messages ...string) error {
	socketAddr := &net.UnixAddr{
		Name: os.Getenv("NOTIFY_SOCKET"),
		Net:  "unixgram",
	}

	if socketAddr.Name == "" {
		return nil
	}

	conn, err := net.DialUnix(socketAddr.Net, nil, socketAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	if _, err = conn.Write([]byte(strings.Join(messages, "\n"))); err != nil {
		return err
	}

	return nil
}
