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
	"github.com/corazawaf/coraza-spoa/pkg/logger"
)

func main() {
	defer func() {
		if err := logger.Sync(); err != nil {
			_ = err
		}
	}()

	flag.Parse()
	if err := config.InitConfig(); err != nil {
		panic(err)
	}

	spoa, err := internal.New(&config.C.SPOA)
	if err != nil {
		logger.Fatal(err.Error())
	}

	if err = spoa.Start(); err != nil {
		logger.Fatal(err.Error())
	}
}
