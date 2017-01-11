package accounttype

import "bitbucket.org/atticlab/go-smart-base/xdr"

func GetAll() []xdr.AccountType {
	return []xdr.AccountType{
		xdr.AccountTypeAccountAnonymousUser,
		xdr.AccountTypeAccountRegisteredUser,
		xdr.AccountTypeAccountMerchant,
		xdr.AccountTypeAccountDistributionAgent,
		xdr.AccountTypeAccountSettlementAgent,
		xdr.AccountTypeAccountExchangeAgent,
		xdr.AccountTypeAccountBank,
		xdr.AccountTypeAccountScratchCard,
	}
}
