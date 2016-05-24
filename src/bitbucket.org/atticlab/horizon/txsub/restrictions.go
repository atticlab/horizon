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
func (sub *submitter) VerifyRestrictionsForSender(sender core.Account, receiver core.Account, payment xdr.PaymentOp) error {
	opAsset, err := assets.Code(payment.Asset)
	if err != nil {
		return err
	}

	opAsset = opAsset
	opAmount := int64(payment.Amount)

	if sender.AccountType == xdr.AccountTypeAccountAnonymousUser && opAsset == "EUAH" {
		var stats map[xdr.AccountType]history.AccountStatistics
		err = sub.historyDb.GetStatisticsByAccountAndAsset(&stats, sender.Accountid, opAsset)
		if err != nil {
			return err
		}

		// 1. Check daily outcome
		dailyOutcome := sumDailyOutcome(
			stats,
			xdr.AccountTypeAccountAnonymousUser,
			xdr.AccountTypeAccountRegisteredUser,
			xdr.AccountTypeAccountSettlementAgent,
		)

		if dailyOutcome+opAmount > sub.config.AnonymousUserRestrictions.MaxDailyOutcome {
			description := fmt.Sprintf(
				"Daily outcoming payments limit for anonymous user exceeded: %s + %s out of %s UAH per day",
				amount.String(xdr.Int64(dailyOutcome)),
				amount.String(payment.Amount),
				amount.String(xdr.Int64(sub.config.AnonymousUserRestrictions.MaxDailyOutcome)),
			)
			return &ExceededLimitError{Description: description}
		}

		// 2. Check monthly outcome
		monthlyOutcome := sumMonthlyOutcome(
			stats,
			xdr.AccountTypeAccountAnonymousUser,
			xdr.AccountTypeAccountRegisteredUser,
			xdr.AccountTypeAccountSettlementAgent,
		)

		if monthlyOutcome+opAmount > sub.config.AnonymousUserRestrictions.MaxMonthlyOutcome {
			description := fmt.Sprintf(
				"Monthly outcoming payments limit for anonymous user exceeded: %s + %s out of %s UAH per month",
				amount.String(xdr.Int64(monthlyOutcome)),
				amount.String(payment.Amount),
				amount.String(xdr.Int64(sub.config.AnonymousUserRestrictions.MaxMonthlyOutcome)),
			)
			return &ExceededLimitError{Description: description}
		}

		// 3. Check annual outcome
		annualOutcome := sumAnnualOutcome(
			stats,
			xdr.AccountTypeAccountAnonymousUser,
			xdr.AccountTypeAccountRegisteredUser,
			xdr.AccountTypeAccountMerchant,
		)

		if annualOutcome+opAmount > sub.config.AnonymousUserRestrictions.MaxAnnualOutcome {
			description := fmt.Sprintf(
				"Annual outcoming payments limit for anonymous user exceeded: %s + %s out of %s UAH per year",
				amount.String(xdr.Int64(annualOutcome)),
				amount.String(payment.Amount),
				amount.String(xdr.Int64(sub.config.AnonymousUserRestrictions.MaxAnnualOutcome)),
			)
			return &ExceededLimitError{Description: description}
		}
	}

	return err
}

// VerifyRestrictionsForReceiver checks limits  and restrictions for receiver
func (sub *submitter) VerifyRestrictionsForReceiver(sender core.Account, receiver core.Account, payment xdr.PaymentOp) error {
	opAsset, err := assets.Code(payment.Asset)
	if err != nil {
		return err
	}

	opAsset = opAsset
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
		var stats map[xdr.AccountType]history.AccountStatistics
		err = sub.historyDb.GetStatisticsByAccountAndAsset(&stats, receiver.Accountid, opAsset)
		if err != nil {
			return err
		}

		annualIncome := sumAnnualIncome(stats)

		if annualIncome+opAmount > sub.config.AnonymousUserRestrictions.MaxAnnualIncome {
			description := fmt.Sprintf(
				"Anonymous user's max annual income limit exceeded: %s + %s out of %s UAH per year",
				amount.String(xdr.Int64(annualIncome)),
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

func sumDailyOutcome(stats map[xdr.AccountType]history.AccountStatistics, args ...xdr.AccountType) int64 {
	var sum int64
	for _, accType := range args {
		println(xdr.AccountType(accType).String())
		if acc, ok := stats[xdr.AccountType(accType)]; ok {
			sum = sum + acc.DailyOutcome
		}
	}

	return sum
}

func sumMonthlyOutcome(stats map[xdr.AccountType]history.AccountStatistics, args ...xdr.AccountType) int64 {
	var sum int64
	for _, accType := range args {
		if acc, ok := stats[xdr.AccountType(accType)]; ok {
			sum = sum + acc.MonthlyOutcome
		}
	}

	return sum
}

func sumAnnualOutcome(stats map[xdr.AccountType]history.AccountStatistics, args ...xdr.AccountType) int64 {
	var sum int64
	for _, accType := range args {
		if acc, ok := stats[xdr.AccountType(accType)]; ok {
			sum = sum + acc.AnnualOutcome
		}
	}

	return sum
}

func sumAnnualIncome(stats map[xdr.AccountType]history.AccountStatistics) int64 {
	var sum int64
	for _, value := range stats {
		sum = sum + value.AnnualIncome
	}

	return sum
}
