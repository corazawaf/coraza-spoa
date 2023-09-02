package server

import (
	"fmt"
	"sync/atomic"
	"unsafe"

	"github.com/corazawaf/coraza-spoa/internal/config"
	"github.com/spf13/viper"
)

var apps unsafe.Pointer

type appManager struct {
	config *viper.Viper
	apps   map[string]*application
}

func (a *appManager) Add(app *config.Application) error {
	if _, ok := a.apps[app.Name]; ok {
		return fmt.Errorf("app %s already exist", app.Name)
	}
	newApp, err := newApplication(app)
	if err != nil {
		return err
	}
	a.apps[app.Name] = newApp
	return nil
}

func (a *appManager) Get(name string) *application {
	return a.apps[name]
}

func newAppManager() *appManager {
	return &appManager{
		config: viper.New(),
		apps:   map[string]*application{},
	}
}

func getApps() *appManager {
	manager := atomic.LoadPointer(&apps)
	return (*appManager)(manager)
}

func setApps(manager *appManager) {
	atomic.StorePointer(&apps, unsafe.Pointer(manager))
}
