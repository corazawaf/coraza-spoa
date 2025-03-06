// Copyright The OWASP Coraza contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"flag"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"syscall"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog"

	"github.com/corazawaf/coraza-spoa/internal"
)

var configPath string
var autoReload bool
var cpuProfile string
var memProfile string
var globalLogger = zerolog.New(os.Stderr).With().Timestamp().Logger()

func main() {
	flag.StringVar(&configPath, "config", "", "configuration file")
	flag.BoolVar(&autoReload, "autoreload", false, "reload configuration file on k8s configmap update")
	flag.StringVar(&cpuProfile, "cpuprofile", "", "write cpu profile to `file`")
	flag.StringVar(&memProfile, "memprofile", "", "write memory profile to `file`")
	flag.Parse()

	if configPath == "" {
		globalLogger.Fatal().Msg("Configuration file is not set")
	}

	if cpuProfile != "" {
		f, err := os.Create(cpuProfile)
		if err != nil {
			globalLogger.Fatal().Err(err).Msg("Could not create CPU profile")
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			globalLogger.Fatal().Err(err).Msg("Could not start CPU profile")
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
			globalLogger.Fatal().Err(err).Msg("Listener closed")
		}
	}()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		globalLogger.Fatal().Err(err).Msg("Failed to create fsnotify watcher")
	}
	defer watcher.Close()

	// configmap mounts are symlinks
	// so we have to watch the parent directory instead of the file itself
	configDir := filepath.Dir(configPath)
	err = watcher.Add(configDir)
	if err != nil {
		globalLogger.Fatal().Err(err).Msg("Failed to add config directory to fsnotify watcher")
	}

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				// on configmap change, the directory symlink is recreated
				// so we have to catch this event and readd the directory back to watcher
				if event.Op == fsnotify.Remove {
					globalLogger.Info().Msg("Config directory updated, reloading configuration...")
					err = watcher.Remove(configDir)
					if err != nil {
						globalLogger.Fatal().Err(err).Msg("Failed to remove config directory from fsnotify watcher")
					}
					err = watcher.Add(configDir)
					if err != nil {
						globalLogger.Fatal().Err(err).Msg("Failed to add config directory to fsnotify watcher")
					}
					newCfg, err := cfg.reloadConfig(a)
					if err != nil {
						globalLogger.Error().Err(err).Msg("Failed to reload configuration, using old configuration")
						continue
					}
					cfg = newCfg
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				globalLogger.Error().Err(err).Msg("Error watching config directory")
			}
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
			newCfg, err := cfg.reloadConfig(a)
			if err != nil {
				globalLogger.Error().Err(err).Msg("Failed to reload configuration, using old configuration")
				continue
			}
			cfg = newCfg
		}
	}

	if memProfile != "" {
		f, err := os.Create(memProfile)
		if err != nil {
			globalLogger.Fatal().Err(err).Msg("Could not create memory profile")
		}
		defer f.Close()
		runtime.GC()
		if err := pprof.WriteHeapProfile(f); err != nil {
			globalLogger.Fatal().Err(err).Msg("Could not write memory profile")
		}
	}
}
