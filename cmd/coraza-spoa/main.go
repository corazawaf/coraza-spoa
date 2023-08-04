// Copyright The OWASP Coraza contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"

	"github.com/corazawaf/coraza-spoa/config"
	"github.com/corazawaf/coraza-spoa/internal"
	"github.com/corazawaf/coraza-spoa/log"
)

func main() {
	cfg := flag.String("config", "", "configuration file")
	//nolint:staticcheck // That's exactly nil check
	if cfg == nil {
		log.Fatal().Msg("configuration file is not set")
	}
	debug := flag.Bool("debug", false, "sets log level to debug")

	flag.Parse()

	log.SetDebug(*debug)

	//nolint:staticcheck // Nil is checked above
	if err := config.InitConfig(*cfg); err != nil {
		log.Fatal().Err(err).Msg("Can't initialize configuration")
	}

	log.InitLogging(config.Global.Log.File, config.Global.Log.Level, config.Global.Log.SpoeLevel)

	spoa, err := internal.New(config.Global)
	if err != nil {
		log.Fatal().Err(err).Msg("Can't initialize SPOA")
	}
	if err := spoa.Start(config.Global.Bind); err != nil {
		log.Fatal().Err(err).Msg("Can't start SPOA")
	}
}
