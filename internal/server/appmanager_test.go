package server

import (
	"testing"

	"github.com/corazawaf/coraza-spoa/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestAppManager(t *testing.T) {
	manager := newAppManager()
	setApps(manager)
	assert.NoError(t, manager.Add(&config.Application{
		Name: "test",
	}))

	manager = getApps()
	assert.NotNil(t, manager.Get("test"))
}
