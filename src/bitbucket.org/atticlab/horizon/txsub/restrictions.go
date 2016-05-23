package txsub

import (
	"fmt"

	"bitbucket.org/atticlab/go-smart-base/amount"
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/assets"
	"bitbucket.org/atticlab/horizon/db2/core"
	"bitbucket.org/atticlab/horizon/db2/history"
)

// VerifyAccountTypesForPayment performs account types check for payment operation
func VerifyAccountTypesForPayment(from core.Account, to core.Account) error {
	if !contains(typeRestrictions[from.AccountType], to.AccountType) {
		reason := fmt.Sprintf("Payments from %s to %s are restricted.", from.AccountType.String(), to.AccountType.String())
		return &RestrictedForAccountTypeError{Reason: reason}
	}

	return nil
}

// VerifyRestrictionsForSender checks limits  and restrictions for sender
func (sub *submitter) VerifyRestrictionsForSender(sender core.Account, payment xdr.PaymentOp) error {
	opAsset, err := assets.Code(payment.Asset)
	if err != nil {
		return err
	}
	opAmount := int64(payment.Amount)

	if sender.AccountType == xdr.AccountTypeAccountAnonymousUser && opAsset == "EUAH" {
		var stats history.AccountStatistics
		err = sub.historyDb.StatisticsByAccountAndAsset(&stats, sender.Accountid, opAsset)
		if err != nil {
			return err
		}

		if stats.DailyOutcome+opAmount > sub.config.AnonymousUserRestrictions.MaxDailyOutcome {
			description := fmt.Sprintf(
				"Daily outcoming payments limit for anonymous user exceeded: %s + %s out of %s UAH per day",
				amount.String(xdr.Int64(stats.DailyOutcome)),
				amount.String(payment.Amount),
				amount.String(xdr.Int64(sub.config.AnonymousUserRestrictions.MaxDailyOutcome)),
			)
			return &ExceededLimitError{Description: description}
		} else if stats.MonthlyOutcome+opAmount > sub.config.AnonymousUserRestrictions.MaxMonthlyOutcome {
			description := fmt.Sprintf(
				"Monthly outcoming payments limit for anonymous user exceeded: %s + %s out of %s UAH per month",
				amount.String(xdr.Int64(stats.MonthlyOutcome)),
				amount.String(payment.Amount),
				amount.String(xdr.Int64(sub.config.AnonymousUserRestrictions.MaxMonthlyOutcome)),
			)
			return &ExceededLimitError{Description: description}
		} else if stats.AnnualOutcome+opAmount > sub.config.AnonymousUserRestrictions.MaxAnnualOutcome {
			description := fmt.Sprintf(
				"Annual outcoming payments limit for anonymous user exceeded: %s + %s out of %s UAH per year",
				amount.String(xdr.Int64(stats.AnnualOutcome)),
				amount.String(payment.Amount),
				amount.String(xdr.Int64(sub.config.AnonymousUserRestrictions.MaxAnnualOutcome)),
			)
			return &ExceededLimitError{Description: description}
		}
	}

	return err
}

// VerifyRestrictionsForReceiver checks limits  and restrictions for receiver
func (sub *submitter) VerifyRestrictionsForReceiver(receiver core.Account, payment xdr.PaymentOp) error {
	opAsset, err := assets.Code(payment.Asset)
	if err != nil {
		return err
	}
	opAmount := int64(payment.Amount)

	if receiver.AccountType == xdr.AccountTypeAccountAnonymousUser && opAsset == "EUAH" {
		// 1. Check max balance
		var trustline core.Trustline
		err = sub.coreDb.TrustlineByAddressAndAsset(&trustline, receiver.Accountid, opAsset, sub.config.BankMasterKey)
		if err != nil {
			return err
		}

		if int64(trustline.Balance)+opAmount > sub.config.AnonymousUserRestrictions.MaxBalance {
			description := fmt.Sprintf(
				"Anonymous user's max balance exceeded: %s + %s out of %s UAH.",
				amount.String(trustline.Balance),
				amount.String(payment.Amount),
				amount.String(xdr.Int64(sub.config.AnonymousUserRestrictions.MaxBalance)),
			)
			return &ExceededLimitError{Description: description}
		}

		// 2.Check max annual income
		var stats history.AccountStatistics
		println(receiver.Accountid)
		err = sub.historyDb.StatisticsByAccountAndAsset(&stats, receiver.Accountid, opAsset)
		if err != nil {
			return err
		}

		if stats.AnnualIncome+opAmount > sub.config.AnonymousUserRestrictions.MaxAnnualIncome {
			description := fmt.Sprintf(
				"Anonymous user's max annual income limit exceeded: %s + %s out of %s UAH per year",
				amount.String(xdr.Int64(stats.AnnualIncome)),
				amount.String(payment.Amount),
				amount.String(xdr.Int64(sub.config.AnonymousUserRestrictions.MaxAnnualIncome)),
			)
			return &ExceededLimitError{Description: description}
		}
	}

	return nil
}

// TODO: generate from template?
// TODO: use sets instead of arrays
// typeRestrictions describes who can send payments to whom
var typeRestrictions = map[xdr.AccountType][]xdr.AccountType{

	xdr.AccountTypeAccountBank: []xdr.AccountType{
		xdr.AccountTypeAccountSettlementAgent,
		xdr.AccountTypeAccountDistributionAgent,
	},

	xdr.AccountTypeAccountDistributionAgent: []xdr.AccountType{
		xdr.AccountTypeAccountAnonymousUser,
		xdr.AccountTypeAccountRegisteredUser,
		xdr.AccountTypeAccountMerchant,
	},

	xdr.AccountTypeAccountSettlementAgent: []xdr.AccountType{
		xdr.AccountTypeAccountBank,
	},

	xdr.AccountTypeAccountExchangeAgent: []xdr.AccountType{},

	xdr.AccountTypeAccountAnonymousUser: []xdr.AccountType{
		xdr.AccountTypeAccountAnonymousUser,
		xdr.AccountTypeAccountRegisteredUser,
		xdr.AccountTypeAccountMerchant,
		xdr.AccountTypeAccountSettlementAgent,
	},

	xdr.AccountTypeAccountRegisteredUser: []xdr.AccountType{
		xdr.AccountTypeAccountAnonymousUser,
		xdr.AccountTypeAccountRegisteredUser,
		xdr.AccountTypeAccountMerchant,
		xdr.AccountTypeAccountSettlementAgent,
	},

	xdr.AccountTypeAccountMerchant: []xdr.AccountType{
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
