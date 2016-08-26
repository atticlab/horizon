// Package cache provides various caches used in horizon.
package cache

import (
	"bitbucket.org/atticlab/horizon/db2/history"
	"github.com/patrickmn/go-cache"
)

// HistoryAccount provides a cached lookup of history_account_id values from
// account addresses.
type HistoryAccount struct {
	*cache.Cache
	q history.QInterface
}

// NewHistoryAccount initializes a new instance of `HistoryAccount`
func NewHistoryAccount(historyQ history.QInterface) *HistoryAccount {
	return &HistoryAccount{
		Cache: cache.New(cache.NoExpiration, cache.NoExpiration),
		q:     historyQ,
	}
}

// Get looks up the History Account ID (i.e. the ID of the operation that
// created the account) for the given strkey encoded address.
func (c *HistoryAccount) Get(address string) (*history.Account, error) {
	found, ok := c.Cache.Get(address)

	if ok {
		result := found.(*history.Account)
		return result, nil
	}

	var rawResult history.Account
	err := c.q.AccountByAddress(&rawResult, address)
	if err != nil {
		return nil, err
	}

	result := &rawResult
	c.Add(address, result)
	return result, nil
}

// Adds address-id pair into cache
func (c *HistoryAccount) Add(address string, account *history.Account) {
	c.Cache.Set(address, account, cache.DefaultExpiration)
}
