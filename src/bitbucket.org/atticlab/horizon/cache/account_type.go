package cache

import (
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/db2/core"
)

// AccountType provides a cached lookup of history_account_id values from
// account addresses.
type AccountType struct {
	Cache
	q core.QInterface
}

// NewAccountType initializes a new instance of `AccountType`
func NewAccountType(coreQ core.QInterface) *AccountType {
	cache := NewCache(100, nil)
	return &AccountType{
		Cache: *cache,
		q:     coreQ,
	}
}

// Get looks up the History Account ID (i.e. the ID of the operation that
// created the account) for the given strkey encoded address.
func (c *AccountType) Get(address string) (result xdr.AccountType, err error) {
	found, ok := c.cached.Get(address)
	if ok {
		result = found.(xdr.AccountType)
		return
	}

	result, err = c.q.AccountTypeByAddress(address)
	if err != nil {
		return
	}

	c.cached.Add(address, result)
	return
}

// Adds address-id pair into cache
func (c *AccountType) Add(address string, accountType xdr.AccountType) {
	c.cached.Add(address, accountType)
}
