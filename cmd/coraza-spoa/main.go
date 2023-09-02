// Copyright The OWASP Coraza contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"flag"
	"os"

	"github.com/corazawaf/coraza-spoa/internal/server"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	var (
		configFile string
	)
	flag.StringVar(&configFile, "f", "/etc/coraza-spoa/config.yaml", "Configuration file")
	flag.Parse()
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	if err := server.Load(configFile); err != nil {
		log.Error().Err(err).Msg("Error loading configuration")
		os.Exit(1)
	}
	if err := server.Serve(context.TODO(), "127.0.0.1:9000", 5); err != nil {
		panic(err)
	}
}
