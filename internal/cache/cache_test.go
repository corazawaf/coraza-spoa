package cache

import (
	"testing"
	"time"

	"github.com/corazawaf/coraza/v3"
	"github.com/stretchr/testify/assert"
)

func TestCache(t *testing.T) {
	waf, err := coraza.NewWAF(coraza.NewWAFConfig())
	assert.NoError(t, err)
	tx := waf.NewTransaction()
	Add(tx, time.Second*1)
	txNew, ok := Get(tx.ID())
	assert.True(t, ok)
	assert.Equal(t, tx.ID(), txNew.ID())
	time.Sleep(5 * time.Second)
	_, ok = Get(tx.ID())
	assert.False(t, ok, "transaction should be removed from cache")
	assert.Equal(t, internalCache.Stats().Evictions, uint64(1), "cache should be evicted")
	// There is a small chance that the cache is not evicted in time
}
