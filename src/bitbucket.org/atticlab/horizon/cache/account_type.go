package cache

import (
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/db2/core"
	"github.com/patrickmn/go-cache"
)

// AccountType provides a cached lookup of core.AccountType values from
// account addresses.
type AccountType struct {
	*cache.Cache
	q core.QInterface
}

// NewAccountType initializes a new instance of `AccountType`
func NewAccountType(coreQ core.QInterface) *AccountType {
	return &AccountType{
		Cache: cache.New(cache.NoExpiration, cache.NoExpiration),
		q:     coreQ,
	}
}

// Get looks up the account type for the given strkey encoded address.
func (c *AccountType) Get(address string) (xdr.AccountType, error) {
	found, ok := c.Cache.Get(address)
	if ok {
		result := found.(*xdr.AccountType)
		return *result, nil
	}

	result, err := c.q.AccountTypeByAddress(address)
	if err != nil {
		return result, err
	}

	c.Add(address, result)
	return result, nil
}

// Adds address-accountType pair into cache
func (c *AccountType) Add(address string, accountType xdr.AccountType) {
	c.Cache.Set(address, &accountType, cache.DefaultExpiration)
}
