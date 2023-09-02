package server

import (
	"errors"
	"fmt"
	"os"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/corazawaf/coraza-spoa/internal/config"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func Load(file string) error {
	log.Info().Str("file", file).Msg("Loading configuration")
	if _, err := os.Stat(file); errors.Is(err, os.ErrNotExist) {
		return err
	}
	manager := newAppManager()
	mainConfig := manager.config
	mainConfig.SetConfigType("yaml")
	mainConfig.SetConfigFile(file)
	if err := mainConfig.ReadInConfig(); err != nil {
		return err
	}
	applications := []*config.Application{}
	// now we load all included files
	for _, include := range mainConfig.GetStringSlice("include") {
		// now we load include as a blob
		// TODO: windows support. Probably will never happen.
		ls, err := doublestar.Glob(os.DirFS("/"), include)
		if err != nil {
			return err
		}
		for _, file := range ls {
			newConfig := viper.New()
			newConfig.SetConfigFile(file)
			if err := mainConfig.MergeConfigMap(newConfig.AllSettings()); err != nil {
				return err
			}
			newApps := []*config.Application{}
			if err := mainConfig.UnmarshalKey("applications", &newApps); err != nil {
				return err
			}
			applications = append(applications, newApps...)
		}
	}
	log.Info().Str("file", file).Msg("Loading default applications")
	globalApps := []*config.Application{}
	if err := mainConfig.UnmarshalKey("applications", &globalApps); err != nil {
		return fmt.Errorf("error unmarshaling configuration: %v", err)
	}
	applications = append(applications, globalApps...)
	for _, app := range applications {
		log.Info().Str("app", app.Name).Msg("Processing application")
		if err := manager.Add(app); err != nil {
			return err
		}
	}
	log.Debug().Int("loaded_apps", len(applications)).Msg("Loaded applications")
	setApps(manager)
	log.Info().Msg("Application Manager pointer sucessfuly overwritten")
	return nil
}

func init() {
	setApps(newAppManager())
}
