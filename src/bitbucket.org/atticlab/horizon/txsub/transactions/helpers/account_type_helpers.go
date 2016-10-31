package helpers

import (
	"bitbucket.org/atticlab/go-smart-base/xdr"
)

// bankAgent returns true if specified user type is a bank agent
func IsBankOrAgent(accountType xdr.AccountType) bool {
	switch accountType {
	case xdr.AccountTypeAccountDistributionAgent, xdr.AccountTypeAccountSettlementAgent, xdr.AccountTypeAccountExchangeAgent, xdr.AccountTypeAccountBank, xdr.AccountTypeAccountGeneralAgent:
		return true
	}
	return false
}

func IsUser(accountType xdr.AccountType) bool {
	switch accountType {
	case xdr.AccountTypeAccountAnonymousUser, xdr.AccountTypeAccountRegisteredUser:
		return true
	}
	return false
}
