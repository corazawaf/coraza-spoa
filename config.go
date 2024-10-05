package main

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"time"

	"github.com/rs/zerolog"
	"gopkg.in/yaml.v3"

	"github.com/corazawaf/coraza-spoa/internal"
)

func readConfig() (*config, error) {
	open, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer open.Close()

	d := yaml.NewDecoder(open)
	d.KnownFields(true)

	var cfg config
	if err := d.Decode(&cfg); err != nil {
		return nil, err
	}

	if len(cfg.Applications) == 0 {
		globalLogger.Warn().Msg("no applications defined")
	}

	return &cfg, nil
}

type config struct {
	Bind         string    `yaml:"bind"`
	Log          logConfig `yaml:",inline"`
	Applications []struct {
		Log                          logConfig `yaml:",inline"`
		Name                         string    `yaml:"name"`
		Directives                   string    `yaml:"directives"`
		ResponseCheck                bool      `yaml:"response_check"`
		TransactionTTLMS             int       `yaml:"transaction_ttl_ms"`
		TransactionActiveLimit       int       `yaml:"transaction_active_limit"`
		TransactionActiveLimitReject bool      `yaml:"transaction_active_limit_reject"`
	} `yaml:"applications"`
}

func (c config) networkAddressFromBind() (network string, address string) {
	bindUrl, err := url.Parse(c.Bind)
	if err == nil {
		return bindUrl.Scheme, bindUrl.Path
	}

	return "tcp", c.Bind
}

func (c config) newApplications() (map[string]*internal.Application, error) {
	allApps := make(map[string]*internal.Application)

	for name, a := range c.Applications {
		logger, err := a.Log.newLogger()
		if err != nil {
			return nil, fmt.Errorf("creating logger for application %q: %v", name, err)
		}

		appConfig := internal.AppConfig{
			Logger:         logger,
			Directives:     a.Directives,
			ResponseCheck:  a.ResponseCheck,
			TransactionTTL: time.Duration(a.TransactionTTLMS) * time.Millisecond,
		}

		application, err := appConfig.NewApplication()
		if err != nil {
			return nil, fmt.Errorf("initializing application %q: %v", name, err)
		}

		allApps[a.Name] = application
	}

	return allApps, nil
}

type logConfig struct {
	Level  string `yaml:"log_level"`
	File   string `yaml:"log_file"`
	Format string `yaml:"log_format"`
}

func (lc logConfig) outputWriter() (io.Writer, error) {
	var out io.Writer
	if lc.File == "" || lc.File == "/dev/stdout" {
		out = os.Stdout
	} else if lc.File == "/dev/stderr" {
		out = os.Stderr
	} else if lc.File == "/dev/null" {
		out = io.Discard
	} else {
		// TODO: Close the handle if not used anymore.
		// Currently these are leaked as soon as we reload.
		f, err := os.OpenFile(lc.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, err
		}
		out = f
	}
	return out, nil
}

func (lc logConfig) newLogger() (zerolog.Logger, error) {
	out, err := lc.outputWriter()
	if err != nil {
		return globalLogger, err
	}

	switch lc.Format {
	case "console":
		out = zerolog.ConsoleWriter{
			Out: out,
		}
	case "json":
	default:
		return globalLogger, fmt.Errorf("unknown log format: %v", lc.Format)
	}

	if lc.Level == "" {
		lc.Level = "info"
	}
	lvl, err := zerolog.ParseLevel(lc.Level)
	if err != nil {
		return globalLogger, err
	}

	return zerolog.New(out).Level(lvl).With().Timestamp().Logger(), nil
}
