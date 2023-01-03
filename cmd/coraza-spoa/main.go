// Copyright The OWASP Coraza contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"

	"github.com/corazawaf/coraza-spoa/config"
	"github.com/corazawaf/coraza-spoa/internal"
)

func main() {
	cfg := flag.String("config", "", "configuration file")
	if cfg == nil {
		panic("configuration file is not set")
	}
	flag.Parse()
	if err := config.InitConfig(*cfg); err != nil {
		panic(err)
	}
	spoa, err := internal.New(config.Global.Applications)
	if err != nil {
		panic(err)
	}
	if err := spoa.Start(config.Global.Bind); err != nil {
		panic(err)
	}
}
