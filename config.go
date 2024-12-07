package main

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
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

	if cfg.DefaultApplication != "" {
		var found bool
		for _, app := range cfg.Applications {
			if app.Name == cfg.DefaultApplication {
				globalLogger.Debug().Str("app", cfg.DefaultApplication).Msg("configured as default application")
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("default application not found among defined applications: %s", cfg.DefaultApplication)
		}
	}

	return &cfg, nil
}

type config struct {
	Bind               string    `yaml:"bind"`
	Log                logConfig `yaml:",inline"`
	DefaultApplication string    `yaml:"default_application"`
	Applications       []struct {
		Log              logConfig `yaml:",inline"`
		Name             string    `yaml:"name"`
		Directives       string    `yaml:"directives"`
		ResponseCheck    bool      `yaml:"response_check"`
		TransactionTTLMS int       `yaml:"transaction_ttl_ms"`
	} `yaml:"applications"`
}

func (c config) networkAddressFromBind() (network string, address string) {
	bindUrl, err := url.Parse(c.Bind)
	if err == nil {
		return bindUrl.Scheme, bindUrl.Path
	}

	return "tcp", c.Bind
}

func (c *config) reloadConfig(a *internal.Agent) (*config, error) {
	newCfg, err := readConfig()
	if err != nil {
		return nil, fmt.Errorf("error loading configuration: %w", err)
	}

	if c.Log != newCfg.Log {
		newLogger, err := newCfg.Log.newLogger()
		if err != nil {
			return nil, fmt.Errorf("error creating new global logger: %w", err)
		}
		globalLogger = newLogger
	}

	if c.Bind != newCfg.Bind {
		return nil, fmt.Errorf("changing bind is not supported yet")
	}

	apps, err := newCfg.newApplications()
	if err != nil {
		return nil, fmt.Errorf("error applying configuration: %w", err)
	}

	a.ReplaceApplications(apps)
	globalLogger.Info().Msg("Configuration successfully reloaded")
	return newCfg, nil
}

func (c *config) watchConfig(a *internal.Agent) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create fsnotify watcher: %w", err)
	}
	defer watcher.Close()

	// configmap mounts are symlinks
	// so we have to watch the parent directory instead of the file itself
	configDir := filepath.Dir(configPath)
	err = watcher.Add(configDir)
	if err != nil {
		return fmt.Errorf("failed to add config directory to fsnotify watcher: %w", err)
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			// on configmap change, the directory symlink is recreated
			// so we have to catch this event and readd the directory back to watcher
			if event.Op == fsnotify.Remove {
				globalLogger.Info().Msg("Config directory updated, reloading configuration...")
				err = watcher.Remove(configDir)
				if err != nil {
					return fmt.Errorf("failed to remove config directory from fsnotify watcher: %w", err)
				}
				err = watcher.Add(configDir)
				if err != nil {
					return fmt.Errorf("failed to add config directory to fsnotify watcher: %w", err)
				}
				newCfg, err := c.reloadConfig(a)
				if err != nil {
					globalLogger.Error().Err(err).Msg("Failed to reload configuration, using old configuration")
					continue
				}
				c = newCfg
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			globalLogger.Error().Err(err).Msg("Error watching config directory")
		}
	}
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
			LogFormat:      a.Log.Format,
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
