// Copyright The OWASP Coraza contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/corazawaf/coraza-spoa/internal/cache"
	"github.com/corazawaf/coraza-spoa/internal/config"
	"github.com/corazawaf/coraza-spoa/internal/logger"
	"github.com/corazawaf/coraza-spoa/internal/server"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	zerolog.TimeFieldFormat = config.DefaultTimeFormat
	var (
		configFile string
	)
	flag.StringVar(&configFile, "f", "/etc/coraza-spoa/config.yaml", "Configuration file")
	flag.Parse()
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	// set zerolog as console
	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: config.DefaultTimeFormat,
	})
	if err := server.Load(configFile); err != nil {
		log.Error().Err(err).Msg("Error loading configuration")
		os.Exit(1)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		if err := server.Serve(ctx, "127.0.0.1:9000", 5); err != nil {
			panic(err)
		}
	}()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGUSR1, syscall.SIGINT)
	for {
		sig := <-sigCh
		switch sig {
		case syscall.SIGTERM:
			log.Info().Msg("Received SIGTERM, shutting down...")
			// this return will run cancel() and close the server
			cache.Shutdown()
			return
		case syscall.SIGINT:
			log.Info().Msg("Received SIGINT, shutting down...")
			return
		case syscall.SIGHUP:
			log.Info().Msg("Received SIGHUP, reloading configuration...")
			if err := server.Load(configFile); err != nil {
				log.Error().Err(err).Msg("Error loading configuration, using old configuration")
			}
		case syscall.SIGUSR1:
			log.Info().Msg("SIGUSR1 received. Changing port is not supported yet")
		}
	}
}

func init() {
	l, err := logger.New("info", "")
	if err != nil {
		panic(fmt.Sprintf("Error creating logger: %v", err))
	}
	log.Logger = *l
	log.Info().Msg("Starting coraza-spoa")
}
