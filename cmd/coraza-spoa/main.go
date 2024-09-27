// Copyright The OWASP Coraza contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/corazawaf/coraza-spoa/config"
	"github.com/corazawaf/coraza-spoa/internal"
)

func main() {
	validate := flag.Bool("validate", false, "validate configuration")
	cfg := flag.String("config", "", "configuration file")
	if cfg == nil {
		panic("configuration file is not set")
	}
	flag.Parse()

	if err := config.InitConfig(*cfg); err != nil {
		panic(err)
	}
	spoa, err := internal.New(config.Global)
	if err != nil {
		panic(err)
	}

	if *validate == true {
		fmt.Println("Configuration is valid")
		os.Exit(0)
	}

	if err := spoa.Start(config.Global.Bind); err != nil {
		panic(err)
	}
}
