package cache

import (
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/go-smart-base/xdr"
)

type accountStatsKey struct {
	Address string
	AssetCode string
	CounterpartyType xdr.AccountType
}

func newAccountStatsKey(address string, assetCode string, counterpartyType xdr.AccountType) accountStatsKey {
	return accountStatsKey{
		Address: address,
		AssetCode: assetCode,
		CounterpartyType: counterpartyType,
	}
}

// AccountStatistics provides a cached lookup of history_account_id values from
// account addresses.
type AccountStatistics struct {
	Cache
	q history.QInterface
}

// NewAccountStatistics initializes a new instance of `AccountStatistics`
func NewAccountStatistics(q history.QInterface) *AccountStatistics {
	cache := NewCache(100, nil)
	return &AccountStatistics{
		Cache: *cache,
		q:     q,
	}
}

// Get looks up the history account statistics for the given strkey encoded address, assetCode and counterparty type.
func (c *AccountStatistics) Get(address string, assetCode string, counterPartyType xdr.AccountType) (result *history.AccountStatistics, err error) {
	key := newAccountStatsKey(address, assetCode, counterPartyType)
	found, ok := c.cached.Get(key)
	if ok {
		result = found.(*history.AccountStatistics)
		return
	}

	stats, err := c.q.GetAccountStatistics(address, assetCode, counterPartyType)
	if err != nil {
		return
	}

	result = &stats
	c.cached.Add(key, result)
	return
}

// Adds address-id pair into cache
func (c *AccountStatistics) Add(stats *history.AccountStatistics) {
	key := newAccountStatsKey(stats.Account, stats.AssetCode, xdr.AccountType(stats.CounterpartyType))
	c.cached.Add(key, stats)
}
