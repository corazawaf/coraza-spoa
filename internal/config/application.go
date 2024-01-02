package config

import "time"

type Application struct {
	// name is used as key to identify the directives
	Name string `json:"name" mapstructure:"name"`

	// directives
	Directives string `json:"directives" mapstructure:"directives"`

	// log level
	LogLevel string `json:"log_level" mapstructure:"log_level"`

	// log file
	LogFile string `json:"log_file" mapstructure:"log_file"`

	ResponseCheck bool `json:"response_check" mapstructure:"response_check"`

	TransactionTTLMs time.Duration `json:"transaction_ttl_ms" mapstructure:"transaction_ttl_ms"`
}
