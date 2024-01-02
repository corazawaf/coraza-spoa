package cache

import (
	"time"

	"github.com/corazawaf/coraza/v3/types"
	"github.com/rs/zerolog/log"
	"istio.io/istio/pkg/cache"
)

var internalCache cache.ExpiringCache

const defaultExpire = time.Second * 10
const defaultEvictionInterval = time.Second * 1

func Add(tx types.Transaction, ttl time.Duration) {
	internalCache.SetWithExpiration(tx.ID(), tx, ttl)
}

func Get(id string) (types.Transaction, bool) {
	tx, ok := internalCache.Get(id)
	if !ok {
		return nil, false
	}
	return tx.(types.Transaction), true
}

// Shutdown forces eviction of all items in the cache
// Important: Shutdown() is not concurrent safe
// It should only be called when the server is shutting down
// And the SPOE server is already shutdown
// Otherwise the server might crash
func Shutdown() {
	internalCache.RemoveAll()
}

func Remove(id string) {
	internalCache.Remove(id)
}

func closeTransaction(tx types.Transaction) {
	// Process Logging won't do anything if TX was already logged.
	tx.ProcessLogging()
	if err := tx.Close(); err != nil {
		log.Error().Err(err).Str("tx", tx.ID()).Msg("error closing transaction")
	}
}

func init() {
	internalCache = cache.NewTTLWithCallback(defaultExpire, defaultEvictionInterval, func(key, value any) {
		// everytime a transaction is timedout we clean it
		tx, ok := value.(types.Transaction)
		if !ok {
			return
		}
		closeTransaction(tx)
	})
}
