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
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Global is used to store the configuration.
var Global *Config

// Config is used to configure coraza-server.
type Config struct {
	Bind         string                  `yaml:"bind"`
	Applications map[string]*Application `yaml:"applications"`
}

// Application is used to manage the haproxy configuration and waf rules.
type Application struct {
	LogLevel               string   `yaml:"log_level"`
	LogFile                string   `yaml:"log_file"`
	Directives             string   `yaml:"directives"`
	Include                []string `yaml:"include"`
	TransactionTTL         int      `yaml:"transaction_ttl"`
	TransactionActiveLimit int      `yaml:"transaction_active_limit"`
}

// InitConfig initializes the configuration.
func InitConfig(file string) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	err = yaml.NewDecoder(f).Decode(&Global)
	if err != nil {
		return err
	}

	// validate the configuration
	err = validateConfig()
	if err != nil {
		return err
	}
	return nil
}

func validateConfig() error {
	for _, app := range Global.Applications {
		if app.LogLevel == "" {
			app.LogLevel = "warn"
		}
		if app.TransactionTTL < 0 {
			return fmt.Errorf("SPOA transaction ttl must be greater than 0")
		}

		if app.TransactionActiveLimit < 0 {
			return fmt.Errorf("SPOA transaction active limit must be greater than 0")
		}
	}
	return nil
}
