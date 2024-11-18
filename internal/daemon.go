package internal

import (
	"net"
	"os"
	"strconv"
	"time"
)

func SdNotify(message string) error {
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

	if _, err = conn.Write([]byte(message)); err != nil {
		return err
	}

	return nil
}

func SdNotifyReady() error {
	return SdNotify("READY=1")
}

func SdNotifyReloading() error {
	microseconds := time.Now().UnixMicro()
	return SdNotify("RELOADING=1\nMONOTONIC_USEC=" + strconv.FormatInt(microseconds, 10))
}

func SdNotifyStopping() error {
	return SdNotify("STOPPING=1")
}
