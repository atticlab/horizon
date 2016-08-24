// Package cache provides various caches used in horizon.
package cache

import (
	"bitbucket.org/atticlab/horizon/db2/history"
	"github.com/patrickmn/go-cache"
)

// HistoryAccount provides a cached lookup of history_account_id values from
// account addresses.
type HistoryAccountID struct {
	*cache.Cache
	q history.QInterface
}

// NewHistoryAccount initializes a new instance of `HistoryAccount`
func NewHistoryAccount(historyQ history.QInterface) *HistoryAccountID {
	return &HistoryAccountID{
		Cache: cache.New(cache.NoExpiration, cache.NoExpiration),
		q:     historyQ,
	}
}

// Get looks up the History Account ID (i.e. the ID of the operation that
// created the account) for the given strkey encoded address.
func (c *HistoryAccountID) Get(address string) (int64, error) {
	found, ok := c.Cache.Get(address)

	if ok {
		result := found.(*int64)
		return *result, nil
	}

	result, err := c.q.AccountIDByAddress(address)
	if err != nil {
		return 0, err
	}

	c.Add(address, result)
	return result, nil
}

// Adds address-id pair into cache
func (c *HistoryAccountID) Add(address string, id int64) {
	c.Cache.Set(address, &id, cache.DefaultExpiration)
}
