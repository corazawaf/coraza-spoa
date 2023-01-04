// Copyright The OWASP Coraza contributors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"os"

	yaml "gopkg.in/yaml.v3"
)

// Global is used to store the configuration.
var Global *Config

// Config is used to configure coraza-server.
type Config struct {
	Bind               string                  `yaml:"bind"`
	DefaultApplication string                  `yaml:"default_application"`
	Applications       map[string]*Application `yaml:"applications"`
}

// Application is used to manage the haproxy configuration and waf rules.
type Application struct {
	LogLevel               string   `yaml:"log_level"`
	LogFile                string   `yaml:"log_file"`
	Rules                  []string `yaml:"rules"`
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
	fmt.Printf("Loading %d applications\n", len(Global.Applications))
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
