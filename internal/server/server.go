package server

import (
	"context"
	"log"
	"net"
	"os"
	"sync/atomic"

	"github.com/negasus/haproxy-spoe-go/logger"
	"github.com/negasus/haproxy-spoe-go/worker"
)

func Serve(ctx context.Context, bind string, maxConnections int) error {
	var connCount int32 = 0 // Atomic counter for connection limiting

	listener, err := net.Listen("tcp4", bind)
	if err != nil {
		log.Printf("error create listener, %v", err)
		os.Exit(1)
	}
	defer listener.Close()

	// Goroutine for handling shutdown
	go func() {
		<-ctx.Done()
		listener.Close()
	}()

	handler := handler{}
	// TODO: set proper logger
	l := logger.NewDefaultLog()
	for {
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				// Context has been canceled, so we're in the process of shutting down
				break
			}
			log.Printf("error accepting connection: %v", err)
			continue
		}

		// Connection limiting logic
		if atomic.AddInt32(&connCount, 1) > int32(maxConnections) {
			conn.Close()
			atomic.AddInt32(&connCount, -1)
			continue
		}

		go func(c net.Conn) {
			defer func() {
				c.Close()
				atomic.AddInt32(&connCount, -1)
			}()

			// Pass connection to the agent
			worker.Handle(c, handler.Handler, l)

		}(conn)
	}

	// Additional cleanup or waiting logic could be added here

	return nil
}
