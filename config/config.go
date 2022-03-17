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

package config

import (
	"flag"
	"fmt"
	"github.com/corazawaf/coraza-spoa/pkg/logger"
	"gopkg.in/yaml.v3"
	"os"
)

// C is used to store the configuration.
var C Config

func init() {
	flag.StringVar(&C.ConfigFile, "config-file", "./config.yml", "The configuration file of the coraza-spoa. (default: ./config.yml)")
}

// Config is used to configure coraza-server.
type Config struct {
	Log  Log  `yaml:"log"`
	SPOA SPOA `yaml:"spoa"`

	// ConfigFile is the configuration file of the coraza-server.
	ConfigFile string
}

// Log is used to configure the level and dir of the log.
type Log struct {
	Level string `yaml:"level"`
	Dir   string `yaml:"dir"`
}

// SPOA is used to manage the haproxy configuration and waf rules.
type SPOA struct {
	Bind                   string   `yaml:"bind"`
	Include                []string `yaml:"include"`
	TransactionTTL         int      `yaml:"transaction_ttl"`
	TransactionActiveLimit int      `yaml:"transaction_active_limit"`
}

// InitConfig initializes the configuration.
func InitConfig() error {
	f, err := os.Open(C.ConfigFile)
	if err != nil {
		return err
	}
	defer f.Close()

	err = yaml.NewDecoder(f).Decode(&C)
	if err != nil {
		return err
	}

	// set the log configuration
	initLog()

	return nil
}

func initLog() {
	var tops = []logger.TeeOption{
		{
			Filename: fmt.Sprintf("%s/server.log", C.Log.Dir),
			ROpts: logger.RotateOptions{
				MaxSize:    128,
				MaxAge:     7,
				MaxBackups: 30,
				Compress:   true,
			},
			Lef: func(level logger.Level) bool {
				l, err := logger.ParseLevel(C.Log.Level)
				if err != nil {
					panic(err)
				}
				return level >= l && level < logger.ErrorLevel
			},
		},
		{
			Filename: fmt.Sprintf("%s/error.log", C.Log.Dir),
			ROpts: logger.RotateOptions{
				MaxSize:    128,
				MaxAge:     7,
				MaxBackups: 30,
				Compress:   true,
			},
			Lef: func(level logger.Level) bool {
				return level >= logger.ErrorLevel
			},
		},
	}

	// reset default logger for using global logger
	logger.NewTeeWithRotate(tops).Reset()
}
