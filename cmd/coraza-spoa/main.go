// Copyright The OWASP Coraza contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"flag"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog/log"

	"github.com/corazawaf/coraza-spoa/internal"
)

func main() {
	flag.Parse()

	log.Info().Msg("Starting coraza-spoa")
	//TODO START HERE

	l, err := net.Listen("tcp", "127.0.0.1:8000")
	if err != nil {
		return
	}

	a := &internal.Agent{
		Context: context.Background(),
		Applications: map[string]*internal.Application{
			"default": {
				ResponseCheck:    true,
				TransactionTTLMs: 1000,
			},
		},
	}

	log.Print(a.Serve(l))

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGUSR1, syscall.SIGINT)
	for {
		sig := <-sigCh
		switch sig {
		case syscall.SIGTERM:
			log.Info().Msg("Received SIGTERM, shutting down...")
			// this return will run cancel() and close the server
			return
		case syscall.SIGINT:
			log.Info().Msg("Received SIGINT, shutting down...")
			return
		case syscall.SIGHUP:
			log.Info().Msg("Received SIGHUP, reloading configuration...")
			log.Error().Err(nil).Msg("Error loading configuration, using old configuration")
		case syscall.SIGUSR1:
			log.Info().Msg("SIGUSR1 received. Changing port is not supported yet")
		}
	}
}
