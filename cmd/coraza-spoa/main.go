// Copyright 2022 The Corazawaf Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
	spoa, err := internal.New(config.Global)
	if err != nil {
		panic(err)
	}
	if err := spoa.Start(config.Global.Bind); err != nil {
		panic(err)
	}
}
