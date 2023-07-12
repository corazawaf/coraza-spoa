// Copyright The OWASP Coraza contributors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"os"

	"github.com/corazawaf/coraza-spoa/log"
	yaml "gopkg.in/yaml.v3"
)

// Global is used to store the configuration.
var Global *Config

// Config is used to configure coraza-server.
type Config struct {
	Bind               string                  `yaml:"bind"`
	Log                Log                     `yaml:"log"`
	DefaultApplication string                  `yaml:"default_application"`
	Applications       map[string]*Application `yaml:"applications"`
}

// Application is used to manage the haproxy configuration and waf rules.
type Application struct {
	// Deprecated: #70: use Config.Log.Level to set up application logging or SecDebugLogLevel to set up Coraza logging
	LogLevel string `yaml:"log_level"`
	// Deprecated: #70: use Config.Log.File to set up application logging or SecDebugLog to set up Coraza logging
	LogFile string `yaml:"log_file"`

	NoResponseCheck bool   `yaml:"no_response_check"`
	Directives      string `yaml:"directives"`
	// Deprecated: use directives instead, this will be removed in the near future.
	Rules                      []string `yaml:"rules"`
	TransactionTTLMilliseconds int      `yaml:"transaction_ttl_ms"`
	TransactionActiveLimit     int      `yaml:"transaction_active_limit"`
}

// Log is used to manage the SPOA logging.
type Log struct {
	Level string `yaml:"level"`
	File  string `yaml:"file"`
	Waf   bool   `yaml:"waf"`
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
	log.Info().Msgf("Loading %d applications", len(Global.Applications))

	for name, app := range Global.Applications {
		log.Debug().Msgf("Validating %s application config", name)

		// Deprecated: #70: use Config.Log.Level to set up application logging or SecDebugLogLevel to set up Coraza logging
		if app.LogLevel != "" {
			log.Warn().Msg("'app.log_level' is skipped. For setting application log level use 'log.level' in the root of configuration.")
		}
		// Deprecated: #70: use Config.Log.File to set up application logging or SecDebugLog to set up Coraza logging
		if app.LogFile != "" {
			log.Warn().Msg("'app.log_level' is skipped. For setting application log file use 'log.file' in the root of configuration.")
		}

		if app.TransactionTTLMilliseconds < 0 {
			return fmt.Errorf("SPOA transaction ttl must be greater than 0")
		}

		if app.TransactionActiveLimit < 0 {
			return fmt.Errorf("SPOA transaction active limit must be greater than 0")
		}
	}
	return nil
}
