// Copyright The OWASP Coraza contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"

	"github.com/corazawaf/coraza-spoa/internal"
)

var (
	version = "dev"
)

var (
	configPath     string
	validateConfig bool
	autoReload     bool
	cpuProfile     string
	memProfile     string
	metricsAddr    string
	healthAddr     string
	showVersion    bool
	globalLogger   = zerolog.New(os.Stderr).With().Timestamp().Logger()
)

func main() {
	flag.StringVar(&configPath, "config", "", "configuration file")
	flag.BoolVar(&validateConfig, "validate", false, "validate configuration file and exit")
	flag.BoolVar(&autoReload, "autoreload", false, "reload configuration file on k8s configmap update")
	flag.StringVar(&cpuProfile, "cpuprofile", "", "write cpu profile to `file`")
	flag.StringVar(&memProfile, "memprofile", "", "write memory profile to `file`")
	flag.StringVar(&metricsAddr, "metrics-addr", "", "ip:port bind for prometheus metrics")
	flag.StringVar(&healthAddr, "health-addr", "", "ip:port bind for health checks")
	flag.BoolVar(&showVersion, "version", false, "show version and exit")
	flag.Parse()

	if showVersion {
		fmt.Printf("version\t%s\n", version)
		if bi, ok := debug.ReadBuildInfo(); ok {
			fmt.Printf("%s", bi.String())
		}
		return
	}

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

	if validateConfig {
		globalLogger.Info().Msg("Configuration file is valid")
		return
	}

	// Create a root context that is canceled on SIGINT or SIGTERM
	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	
	// Create a child context for the agent that can be canceled independently
	rootCtx, rootCancel := context.WithCancel(rootCtx)
	defer rootCancel()

	// Listen for SIGHUP to trigger configuration reload
	reloadCh := make(chan os.Signal, 1)
	signal.Notify(reloadCh, syscall.SIGHUP)

	agent := &internal.Agent{
		Context:            rootCtx,
		DefaultApplication: apps[cfg.DefaultApplication],
		Applications:       apps,
		Logger:             globalLogger,
	}
	
	// Start the agent in a separate goroutine
	go runAgent(rootCtx, cfg, agent, rootCancel)
	
	if metricsAddr != "" {
		go runMetricsServer(rootCtx, metricsAddr)
	}

	if healthAddr != "" {
		go runHealthServer(rootCtx, healthAddr)
	}

	if autoReload {
		go func() {
			if err := cfg.watchConfig(agent); err != nil {
				globalLogger.Fatal().Err(err).Msg("Config watcher failed")
			}
		}()
	}

outer:
	for {
		select {
		case <-rootCtx.Done():
			globalLogger.Info().Msg("Received SIGTERM/SIGINT, shutting down...")
			break outer
		case <-reloadCh:
			globalLogger.Info().Msg("Received SIGHUP, reloading configuration...")
			newCfg, err := cfg.reloadConfig(agent)
			if err != nil {
				globalLogger.Error().Err(err).Msg("Failed to reload configuration, using old configuration")
				continue
			}
			cfg = newCfg
		}
	}

	// Drain in-flight detect-only background evaluations before exit.
	agent.DrainDetectOnly()

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

func runAgent(ctx context.Context, cfg *config, agent *internal.Agent, rootCancel context.CancelFunc) {
	network, address := cfg.networkAddressFromBind()
	l, err := (&net.ListenConfig{}).Listen(ctx, network, address)
	if err != nil {
		globalLogger.Fatal().Err(err).Msg("Failed opening socket")
		rootCancel()
		return
	}


	// Ensure the listener is closed when the context is canceled
	go func() {
		<-ctx.Done()
		if err := l.Close(); err != nil {
			globalLogger.Error().Err(err).Msg("Failed closing listener")
		}
	}()


	globalLogger.Info().Msg("Starting coraza-spoa")
	if err := agent.Serve(l); err != nil {
		select {
		case <-ctx.Done():
			// Listener was closed due to shutdown, ignore error
			// and exit gracefully
			return
		default:
			// Unexpected error, log and exit
		}

		globalLogger.Fatal().Err(err).Msg("Listener closed")
		rootCancel()
	}
}

func runMetricsServer(ctx context.Context, addr string) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		BaseContext: func(net.Listener) context.Context {
			return ctx
		},
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		globalLogger.Error().Err(err).Msg("Metrics server failed")
	}
}

func runHealthServer(ctx context.Context, addr string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("ok")); err != nil {
			globalLogger.Error().Err(err).Msg("Health check response error")
		}
	})

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		BaseContext: func(net.Listener) context.Context {
			return ctx
		},
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		globalLogger.Error().Err(err).Msg("Health server failed")
	}
}
