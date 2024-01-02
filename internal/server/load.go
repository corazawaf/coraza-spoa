package server

import (
	"errors"
	"fmt"
	"os"

	"github.com/corazawaf/coraza-spoa/internal/config"
	"github.com/rs/zerolog/log"
)

func Load(file string) error {
	if _, err := os.Stat(file); errors.Is(err, os.ErrNotExist) {
		return err
	}
	log.Info().Msgf("Loading configurations from %s", file)
	manager := newAppManager()
	mainConfig := manager.config
	mainConfig.SetConfigType("yaml")
	mainConfig.SetConfigFile(file)
	if err := mainConfig.ReadInConfig(); err != nil {
		return err
	}
	if def := mainConfig.GetString("default_application"); def == "" {
		manager.defaultApplication = "default"
	} else {
		manager.defaultApplication = def
	}

	// now we load all included files
	globalApps := []*config.Application{}
	if err := mainConfig.UnmarshalKey("applications", &globalApps); err != nil {
		return fmt.Errorf("error unmarshaling configuration: %v", err)
	}
	for _, app := range globalApps {
		if err := manager.Add(app); err != nil {
			return err
		}
	}
	setApps(manager)
	return nil
}

func init() {
	setApps(newAppManager())
}
