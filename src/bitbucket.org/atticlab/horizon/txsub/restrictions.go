package txsub

import (
    "bitbucket.org/atticlab/go-smart-base/xdr"
    "bitbucket.org/atticlab/horizon/db2/core"
    "fmt"
)

// VerifyAccountTypesForPayment performs account types check for payment operation
func VerifyAccountTypesForPayment(from core.Account, to core.Account) error {
    if (!contains(typeRestrictions[from.AccountType], to.AccountType)) {
        reason := fmt.Sprintf("Payments from %s to %s are restricted.", from.AccountType.String(), to.AccountType.String())
        return &RestrictedForAccountTypeError{ Reason: reason}
    }
    
    return nil 
}

// TODO: generate from template?
// TODO: use sets instead of arrays
// typeRestrictions describes who can send payments to whom
var typeRestrictions = map[xdr.AccountType][]xdr.AccountType {
    
	xdr.AccountTypeAccountBank: []xdr.AccountType {
		xdr.AccountTypeAccountSettlementAgent,
		xdr.AccountTypeAccountDistributionAgent,
	},
    
    xdr.AccountTypeAccountDistributionAgent: []xdr.AccountType {
		xdr.AccountTypeAccountAnonymousUser,
		xdr.AccountTypeAccountRegisteredUser,
		xdr.AccountTypeAccountMerchant,
	},
    
    xdr.AccountTypeAccountSettlementAgent: []xdr.AccountType {
        xdr.AccountTypeAccountBank,
	},
    
    xdr.AccountTypeAccountExchangeAgent: []xdr.AccountType {
	},
    
	xdr.AccountTypeAccountAnonymousUser: []xdr.AccountType {
		xdr.AccountTypeAccountAnonymousUser,
		xdr.AccountTypeAccountRegisteredUser,
		xdr.AccountTypeAccountMerchant,
		xdr.AccountTypeAccountSettlementAgent,
	},
	
	xdr.AccountTypeAccountRegisteredUser: []xdr.AccountType {
		xdr.AccountTypeAccountAnonymousUser,
		xdr.AccountTypeAccountRegisteredUser,
		xdr.AccountTypeAccountMerchant,
		xdr.AccountTypeAccountSettlementAgent,
	},
	
	xdr.AccountTypeAccountMerchant: []xdr.AccountType {
		xdr.AccountTypeAccountAnonymousUser,
		xdr.AccountTypeAccountRegisteredUser,
		xdr.AccountTypeAccountMerchant,
		xdr.AccountTypeAccountSettlementAgent,
	},
}

func contains(list []xdr.AccountType, a xdr.AccountType) bool {
    for _, b := range list {
        if b == a {
            return true
        }
    }
    return false
}
