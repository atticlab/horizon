// Package cache provides various caches used in horizon.
package cache

import (
	"bitbucket.org/atticlab/horizon/db2"
)

// HistoryAccount provides a cached lookup of history_account_id values from
// account addresses.
type HistoryAccount struct {
	Cache
	db *db2.Repo
}

// NewHistoryAccount initializes a new instance of `HistoryAccount`
func NewHistoryAccount(db *db2.Repo) *HistoryAccount {
	cache := NewCache(100, nil)
	return &HistoryAccount{
		Cache: *cache,
		db:    db,
	}
}

// Get looks up the History Account ID (i.e. the ID of the operation that
// created the account) for the given strkey encoded address.
func (c *HistoryAccount) Get(address string) (result int64, err error) {
	found, ok := c.cached.Get(address)

	if ok {
		result = found.(int64)
		return
	}

	err = c.db.GetRaw(&result, `
		SELECT id
		FROM history_accounts
		WHERE address = $1
		ORDER BY id DESC
	`, address)

	if err != nil {
		return
	}

	c.cached.Add(address, result)
	return
}

// Adds address-id pair into cache
func (c *HistoryAccount) Add(address string, id int64) {
	c.cached.Add(address, id)
}
