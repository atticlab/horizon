package validator

import (
	"fmt"

	"database/sql"

	"bitbucket.org/atticlab/go-smart-base/amount"
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/assets"
	"bitbucket.org/atticlab/horizon/db2/core"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/txsub/results"
)

// VerifyAccountTypesForPayment performs account types check for payment operation
func (v *LimitsValidator) verifyAccountTypesForPayment(from core.Account, to core.Account) error {
	if !contains(typeRestrictions[from.AccountType], to.AccountType) {
		reason := fmt.Sprintf("Payments from %s to %s are restricted.", from.AccountType.String(), to.AccountType.String())
		return &results.RestrictedForAccountTypeError{Reason: reason}
	}

	return nil
}

// VerifyRestrictions checks traits of the involved accounts
func (v *LimitsValidator) verifyRestrictions(from string, to string) error {
	// Get account traits
	var sourceTraits, destTraits history.AccountTraits
	errSource := v.historyDb.GetAccountTraitsByAddress(&sourceTraits, from)
	if errSource != nil && errSource != sql.ErrNoRows {
		return errSource
	}

	errDest := v.historyDb.GetAccountTraitsByAddress(&destTraits, to)
	if errDest != nil && errDest != sql.ErrNoRows {
		return errDest
	}

	// Check restrictions
	if errSource != nil && sourceTraits.BlockOutcomingPayments {
		return &results.RestrictedForAccountError{
			Reason:  "Outcoming payments for this account are restricted by administrator.",
			Address: from,
		}
	}
	if errDest != nil && destTraits.BlockIncomingPayments {
		return &results.RestrictedForAccountError{
			Reason:  "Incoming payments for this account are restricted by administrator.",
			Address: to,
		}
	}

	return nil
}

// VerifyLimitsForSender checks limits for sender
func (v *LimitsValidator) verifyLimitsForSender(sender core.Account, receiver core.Account, payment xdr.PaymentOp) error {
	opAmount := int64(payment.Amount)
	opAsset, err := assets.Code(payment.Asset)
	if err != nil {
		return err
	}

	var limits history.AccountLimits
	hasLimits := false
	err = v.historyDb.GetAccountLimits(&limits, sender.Accountid, opAsset)
	if err == nil {
		if limits.MaxOperationOut >= 0 && opAmount > limits.MaxOperationOut*amount.One {
			description := fmt.Sprintf(
				"Maximal operation amount for this account exceeded: %s of %s %s",
				amount.String(payment.Amount),
				amount.String(xdr.Int64(limits.MaxOperationOut*amount.One)),
				opAsset,
			)
			return &results.ExceededLimitError{Description: description}
		}
		hasLimits = true
	}

	if !hasLimits && opAsset != "EUAH" {
		// Nothing to be checked
		return nil
	}

	var stats map[xdr.AccountType]history.AccountStatistics
	err = v.historyDb.GetStatisticsByAccountAndAsset(&stats, sender.Accountid, opAsset)
	if err != nil {
		return err
	}
	dailyOutcome := sumDailyOutcome(
		stats,
		xdr.AccountTypeAccountAnonymousUser,
		xdr.AccountTypeAccountRegisteredUser,
		xdr.AccountTypeAccountSettlementAgent,
	)
	monthlyOutcome := sumMonthlyOutcome(
		stats,
		xdr.AccountTypeAccountAnonymousUser,
		xdr.AccountTypeAccountRegisteredUser,
		xdr.AccountTypeAccountSettlementAgent,
	)

	if hasLimits && limits.DailyMaxOut >= 0 && dailyOutcome+opAmount > limits.DailyMaxOut*amount.One {
		description := fmt.Sprintf(
			"Daily outcoming payments limit for account exceeded: %s + %s out of %s UAH per day",
			amount.String(xdr.Int64(dailyOutcome)),
			amount.String(payment.Amount),
			amount.String(xdr.Int64(limits.DailyMaxOut*amount.One)),
		)
		return &results.ExceededLimitError{Description: description}
	}
	if hasLimits && limits.MonthlyMaxOut >= 0 && monthlyOutcome+opAmount > limits.MonthlyMaxOut*amount.One {
		description := fmt.Sprintf(
			"Monthly outcoming payments limit for account exceeded: %s + %s out of %s UAH per month",
			amount.String(xdr.Int64(monthlyOutcome)),
			amount.String(payment.Amount),
			amount.String(xdr.Int64(limits.MonthlyMaxOut*amount.One)),
		)
		return &results.ExceededLimitError{Description: description}
	}

	if sender.AccountType == xdr.AccountTypeAccountAnonymousUser && opAsset == "EUAH" {
		// 1. Check daily outcome
		if dailyOutcome+opAmount > v.config.AnonymousUserRestrictions.MaxDailyOutcome {
			description := fmt.Sprintf(
				"Daily outcoming payments limit for anonymous user exceeded: %s + %s out of %s UAH per day",
				amount.String(xdr.Int64(dailyOutcome)),
				amount.String(payment.Amount),
				amount.String(xdr.Int64(v.config.AnonymousUserRestrictions.MaxDailyOutcome)),
			)
			return &results.ExceededLimitError{Description: description}
		}

		// 2. Check monthly outcome
		if monthlyOutcome+opAmount > v.config.AnonymousUserRestrictions.MaxMonthlyOutcome {
			description := fmt.Sprintf(
				"Monthly outcoming payments limit for anonymous user exceeded: %s + %s out of %s UAH per month",
				amount.String(xdr.Int64(monthlyOutcome)),
				amount.String(payment.Amount),
				amount.String(xdr.Int64(v.config.AnonymousUserRestrictions.MaxMonthlyOutcome)),
			)
			return &results.ExceededLimitError{Description: description}
		}
	}

	if opAsset == "EUAH" && !bankAgent(sender.AccountType) {
		// 3. Check annual outcome
		annualOutcome := sumAnnualOutcome(
			stats,
			xdr.AccountTypeAccountAnonymousUser,
			xdr.AccountTypeAccountRegisteredUser,
			xdr.AccountTypeAccountMerchant,
		)

		if annualOutcome+opAmount > v.config.AnonymousUserRestrictions.MaxAnnualOutcome {
			description := fmt.Sprintf(
				"Annual outcoming payments limit for user exceeded: %s + %s out of %s UAH per year",
				amount.String(xdr.Int64(annualOutcome)),
				amount.String(payment.Amount),
				amount.String(xdr.Int64(v.config.AnonymousUserRestrictions.MaxAnnualOutcome)),
			)
			return &results.ExceededLimitError{Description: description}
		}
	}

	return err
}

// VerifyLimitsForReceiver checks limits  and restrictions for receiver
func (v *LimitsValidator) verifyLimitsForReceiver(sender core.Account, receiver core.Account, payment xdr.PaymentOp) error {
	opAsset, err := assets.Code(payment.Asset)
	if err != nil {
		return err
	}

	opAmount := int64(payment.Amount)

	var limits history.AccountLimits
	hasLimits := false
	err = v.historyDb.GetAccountLimits(&limits, receiver.Accountid, opAsset)
	if err == nil {
		if limits.MaxOperationIn >= 0 && opAmount > limits.MaxOperationIn*amount.One {
			description := fmt.Sprintf(
				"Maximal income operation amount for this account exceeded: %s of %s %s",
				amount.String(payment.Amount),
				amount.String(xdr.Int64(limits.MaxOperationIn*amount.One)),
				opAsset,
			)
			return &results.ExceededLimitError{Description: description}
		}
		hasLimits = true
	}

	if !hasLimits && opAsset != "EUAH" {
		// Nothing to be checked
		return nil
	}
	var stats map[xdr.AccountType]history.AccountStatistics
	err = v.historyDb.GetStatisticsByAccountAndAsset(&stats, receiver.Accountid, opAsset)
	if err != nil {
		return err
	}

	dailyIncome := sumDailyIncome(
		stats,
		xdr.AccountTypeAccountAnonymousUser,
		xdr.AccountTypeAccountRegisteredUser,
		xdr.AccountTypeAccountMerchant,
		xdr.AccountTypeAccountDistributionAgent,
		xdr.AccountTypeAccountBank,
	)
	monthlyIncome := sumMonthlyIncome(
		stats,
		xdr.AccountTypeAccountAnonymousUser,
		xdr.AccountTypeAccountRegisteredUser,
		xdr.AccountTypeAccountMerchant,
		xdr.AccountTypeAccountDistributionAgent,
		xdr.AccountTypeAccountBank,
	)

	if hasLimits && limits.DailyMaxIn >= 0 && dailyIncome+opAmount > limits.DailyMaxIn*amount.One {
		description := fmt.Sprintf(
			"Daily incoming payments limit for account exceeded: %s + %s out of %s %s per day",
			amount.String(xdr.Int64(dailyIncome)),
			amount.String(payment.Amount),
			amount.String(xdr.Int64(limits.DailyMaxIn*amount.One)),
			opAsset,
		)
		return &results.ExceededLimitError{Description: description}
	}
	if hasLimits && limits.MonthlyMaxIn >= 0 && monthlyIncome+opAmount > limits.MonthlyMaxIn*amount.One {
		description := fmt.Sprintf(
			"Monthly incoming payments limit for account exceeded: %s + %s out of %s %s per month",
			amount.String(xdr.Int64(monthlyIncome)),
			amount.String(payment.Amount),
			amount.String(xdr.Int64(limits.MonthlyMaxIn*amount.One)),
			opAsset,
		)
		return &results.ExceededLimitError{Description: description}
	}

	if opAsset == "EUAH" && !bankAgent(receiver.AccountType) {
		// 1. Check max balance
		var trustline core.Trustline
		err = v.coreDb.TrustlineByAddressAndAsset(&trustline, receiver.Accountid, opAsset, v.config.BankMasterKey)
		if err == sql.ErrNoRows {
			// let's suppose the balance is zero and let core throw error
			trustline.Balance = 0
		} else {
			if err != nil {
				return err
			}
		}

		if int64(trustline.Balance)+opAmount > v.config.AnonymousUserRestrictions.MaxBalance {
			description := fmt.Sprintf(
				"User's max balance exceeded: %s + %s out of %s UAH.",
				amount.String(trustline.Balance),
				amount.String(payment.Amount),
				amount.String(xdr.Int64(v.config.AnonymousUserRestrictions.MaxBalance)),
			)
			return &results.ExceededLimitError{Description: description}
		}

		// 2.Check max annual income
		annualIncome := sumAnnualIncome(stats)

		if annualIncome+opAmount > v.config.AnonymousUserRestrictions.MaxAnnualIncome {
			description := fmt.Sprintf(
				"User's max annual income limit exceeded: %s + %s out of %s UAH per year",
				amount.String(xdr.Int64(annualIncome)),
				amount.String(payment.Amount),
				amount.String(xdr.Int64(v.config.AnonymousUserRestrictions.MaxAnnualIncome)),
			)
			return &results.ExceededLimitError{Description: description}
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

// bankAgent returns true if specified user type is a bank agent
func bankAgent(accountType xdr.AccountType) bool {
	isAgent := accountType != xdr.AccountTypeAccountAnonymousUser
	isAgent = isAgent && accountType != xdr.AccountTypeAccountRegisteredUser
	isAgent = isAgent && accountType != xdr.AccountTypeAccountMerchant

	return isAgent
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

func sumDailyIncome(stats map[xdr.AccountType]history.AccountStatistics, args ...xdr.AccountType) int64 {
	var sum int64
	for _, accType := range args {
		println(xdr.AccountType(accType).String())
		if acc, ok := stats[xdr.AccountType(accType)]; ok {
			sum = sum + acc.DailyIncome
		}
	}

	return sum
}

func sumMonthlyIncome(stats map[xdr.AccountType]history.AccountStatistics, args ...xdr.AccountType) int64 {
	var sum int64
	for _, accType := range args {
		if acc, ok := stats[xdr.AccountType(accType)]; ok {
			sum = sum + acc.MonthlyIncome
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

func contains(list []xdr.AccountType, a xdr.AccountType) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
