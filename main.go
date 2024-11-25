// Copyright The OWASP Coraza contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"flag"
	"net"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"syscall"

	"github.com/rs/zerolog"

	"github.com/corazawaf/coraza-spoa/internal"
)

var configPath string
var cpuProfile string
var memProfile string
var checkMode bool
var globalLogger = zerolog.New(os.Stderr).With().Timestamp().Logger()

func main() {
	flag.StringVar(&cpuProfile, "cpuprofile", "", "write cpu profile to `file`")
	flag.StringVar(&memProfile, "memprofile", "", "write memory profile to `file`")
	flag.StringVar(&configPath, "config", "", "configuration file")
	flag.StringVar(&configPath, "f", "", "configuration file")
	flag.BoolVar(&checkMode, "check", false, "check mode : only check config files and exit")
	flag.BoolVar(&checkMode, "c", false, "check mode : only check config files and exit")
	flag.Parse()

	if configPath == "" {
		globalLogger.Fatal().Msg("Configuration file is not set")
	}

	if cpuProfile != "" {
		f, err := os.Create(cpuProfile)
		if err != nil {
			globalLogger.Fatal().Err(err).Msg("could not create CPU profile")
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			globalLogger.Fatal().Err(err).Msg("could not start CPU profile")
		}
		defer pprof.StopCPUProfile()
	}

	cfg, err := readConfig()
	if err != nil {
		globalLogger.Fatal().Err(err).Msg("Failed loading config")
	}

	logger, err := cfg.Log.newLogger()
	if err != nil {
		globalLogger.Fatal().Err(err).Msg("Failed creating global logger")
	}
	globalLogger = logger

	apps, err := cfg.newApplications()
	if err != nil {
		globalLogger.Fatal().Err(err).Msg("Failed creating applications")
	}

	if checkMode {
		globalLogger.Info().Msg("Configuration file is valid")
		os.Exit(0)
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	network, address := cfg.networkAddressFromBind()
	l, err := (&net.ListenConfig{}).Listen(ctx, network, address)
	if err != nil {
		globalLogger.Fatal().Err(err).Msg("Failed opening socket")
	}

	a := &internal.Agent{
		Context:      ctx,
		Applications: apps,
		Logger:       globalLogger,
	}
	go func() {
		defer cancelFunc()

		globalLogger.Info().Msg("Starting coraza-spoa")
		if err := a.Serve(l); err != nil {
			globalLogger.Fatal().Err(err).Msg("listener closed")
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGUSR1, syscall.SIGINT)
outer:
	for {
		sig := <-sigCh
		switch sig {
		case syscall.SIGTERM:
			globalLogger.Info().Msg("Received SIGTERM, shutting down...")
			// this return will run cancel() and close the server
			break outer
		case syscall.SIGINT:
			globalLogger.Info().Msg("Received SIGINT, shutting down...")
			break outer
		case syscall.SIGHUP:
			globalLogger.Info().Msg("Received SIGHUP, reloading configuration...")

			newCfg, err := readConfig()
			if err != nil {
				globalLogger.Error().Err(err).Msg("Error loading configuration, using old configuration")
				continue
			}

			if cfg.Log != newCfg.Log {
				newLogger, err := newCfg.Log.newLogger()
				if err != nil {
					globalLogger.Error().Err(err).Msg("Error creating new global logger, using old configuration")
					continue
				}
				globalLogger = newLogger
			}

			if cfg.Bind != newCfg.Bind {
				globalLogger.Error().Msg("Changing bind is not supported yet, using old configuration")
				continue
			}

			apps, err := newCfg.newApplications()
			if err != nil {
				globalLogger.Error().Err(err).Msg("Error applying configuration, using old configuration")
				continue
			}

			a.ReplaceApplications(apps)
			cfg = newCfg
		}
	}

	if memProfile != "" {
		f, err := os.Create(memProfile)
		if err != nil {
			globalLogger.Fatal().Err(err).Msg("could not create memory profile")
		}
		defer f.Close()
		runtime.GC()
		if err := pprof.WriteHeapProfile(f); err != nil {
			globalLogger.Fatal().Err(err).Msg("could not write memory profile")
		}
	}
}
