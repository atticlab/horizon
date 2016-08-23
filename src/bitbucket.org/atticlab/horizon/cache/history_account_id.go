// Package cache provides various caches used in horizon.
package cache

import (
	"bitbucket.org/atticlab/horizon/db2/history"
)

// HistoryAccount provides a cached lookup of history_account_id values from
// account addresses.
type HistoryAccountID struct {
	Cache
	q history.QInterface
}

// NewHistoryAccount initializes a new instance of `HistoryAccount`
func NewHistoryAccount(historyQ history.QInterface) *HistoryAccountID {
	cache := NewCache(0, nil)
	return &HistoryAccountID{
		Cache: *cache,
		q:     historyQ,
	}
}

// Get looks up the History Account ID (i.e. the ID of the operation that
// created the account) for the given strkey encoded address.
func (c *HistoryAccountID) Get(address string) (result int64, err error) {
	found, ok := c.cached.Get(address)

	if ok {
		result = found.(int64)
		return
	}

	result, err = c.q.AccountIDByAddress(address)

	if err != nil {
		return
	}

	c.cached.Add(address, result)
	return
}

// Adds address-id pair into cache
func (c *HistoryAccountID) Add(address string, id int64) {
	c.cached.Add(address, id)
}
