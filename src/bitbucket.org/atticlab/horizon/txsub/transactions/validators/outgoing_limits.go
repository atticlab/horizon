package validators

import (
	"bitbucket.org/atticlab/go-smart-base/amount"
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/config"
	"bitbucket.org/atticlab/horizon/db2/core"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/log"
	"bitbucket.org/atticlab/horizon/txsub/results"
	"bitbucket.org/atticlab/horizon/txsub/transactions/helpers"
	"database/sql"
	"fmt"
)

type OutgoingLimitsValidatorInterface interface {
	VerifyLimits() (*results.ExceededLimitError, error)
}

type OutgoingLimitsValidator struct {
	limitsValidator
	dailyOutcome   *int64
	monthlyOutcome *int64
}

func NewOutgoingLimitsValidator(account, counterparty *core.Account, opAmount int64, opAsset history.Asset, historyQ history.QInterface, anonUserRestr config.AnonymousUserRestrictions) *OutgoingLimitsValidator {
	limitsValidator := newLimitsValidator(PaymentTypeOutgoing, account, counterparty, opAmount, opAsset, historyQ, anonUserRestr)
	result := &OutgoingLimitsValidator{
		limitsValidator: *limitsValidator,
	}
	result.log = log.WithField("service", "outgoing_limits_validator")
	return result
}

// VerifyLimits checks outgoing limits
func (v *OutgoingLimitsValidator) VerifyLimits() (*results.ExceededLimitError, error) {
	// check account's limits
	result, err := v.verifySenderAccountLimits()
	if result != nil || err != nil {
		return result, err
	}

	return v.verifyAnonymousAssetLimits()
}

// Checks limits for sender
func (v *OutgoingLimitsValidator) verifySenderAccountLimits() (*results.ExceededLimitError, error) {
	var limits history.AccountLimits
	err := v.historyQ.GetAccountLimits(&limits, v.account.Accountid, v.opAsset.Code)
	if err != nil {
		// no limits to check for sender
		if err == sql.ErrNoRows {
			v.log.Debug("No limits found")
			return nil, nil
		}
		return nil, err
	}

	v.log.WithField("limits", limits).Debug("Checking limits")
	if limits.MaxOperationOut >= 0 && v.opAmount > limits.MaxOperationOut {
		description := fmt.Sprintf(
			"Maximal operation amount for account (%s) exceeded: %s of %s %s",
			v.account.Accountid,
			amount.String(xdr.Int64(v.opAmount)),
			amount.String(xdr.Int64(limits.MaxOperationOut)),
			v.opAsset.Code,
		)
		return &results.ExceededLimitError{Description: description}, nil
	}

	if limits.DailyMaxOut >= 0 {
		dailyOutcome, err := v.getDailyOutcome()
		if err != nil {
			return nil, err
		}
		v.log.WithFields(log.F{
			"newOutcome": dailyOutcome + v.opAmount,
			"limit":      limits.DailyMaxOut,
		}).Debug("Checking daily outcome for limits")
		if dailyOutcome+v.opAmount > limits.DailyMaxOut {
			description := v.limitExceededDescription("Daily", false, dailyOutcome, limits.DailyMaxOut)
			return &results.ExceededLimitError{Description: description}, nil
		}
	}

	if limits.MonthlyMaxOut >= 0 {
		monthlyOutcome, err := v.getMonthlyOutcome()
		if err != nil {
			return nil, err
		}
		v.log.WithFields(log.F{
			"newOutcome": monthlyOutcome + v.opAmount,
			"limit":      limits.MonthlyMaxOut,
		}).Debug("Checking daily outcome for limits")
		if monthlyOutcome+v.opAmount > limits.MonthlyMaxOut {
			description := v.limitExceededDescription("Monthly", false, monthlyOutcome, limits.MonthlyMaxOut)
			return &results.ExceededLimitError{Description: description}, nil
		}
	}
	return nil, nil
}

// checks limits for anonymous asset
func (v *OutgoingLimitsValidator) verifyAnonymousAssetLimits() (*results.ExceededLimitError, error) {
	if !v.opAsset.IsAnonymous || !helpers.IsUser(v.account.AccountType) {
		// Nothing to be checked
		return nil, nil
	}
	// check anonymous asset limits
	// daily and monthly limits are not applied for payments to merchant
	if v.counterparty.AccountType != xdr.AccountTypeAccountMerchant {
		// 1. Check daily outcome
		dailyOutcome, err := v.getDailyOutcome()
		if err != nil {
			return nil, err
		}
		if dailyOutcome+v.opAmount > v.anonUserRest.MaxDailyOutcome {
			description := v.limitExceededDescription("Daily", true, dailyOutcome, v.anonUserRest.MaxDailyOutcome)
			return &results.ExceededLimitError{Description: description}, nil
		}

		// 2. Check monthly outcome
		monthlyOutcome, err := v.getMonthlyOutcome()
		if err != nil {
			return nil, err
		}

		if monthlyOutcome+v.opAmount > v.anonUserRest.MaxMonthlyOutcome {
			description := v.limitExceededDescription("Monthly", true, monthlyOutcome, v.anonUserRest.MaxMonthlyOutcome)
			return &results.ExceededLimitError{Description: description}, nil
		}
	}

	// annualOutcome does not count for payments to settlement agent
	if v.counterparty.AccountType != xdr.AccountTypeAccountSettlementAgent {
		// 3. Check annual outcome
		stats, err := v.getAccountStats()
		if err != nil {
			return nil, err
		}

		annualOutcome := helpers.SumAccountStats(
			stats,
			func(stats *history.AccountStatistics) int64 { return stats.AnnualOutcome },
			xdr.AccountTypeAccountAnonymousUser,
			xdr.AccountTypeAccountRegisteredUser,
			xdr.AccountTypeAccountMerchant,
		)

		if annualOutcome+v.opAmount > v.anonUserRest.MaxAnnualOutcome {
			description := v.limitExceededDescription("Annual", true, annualOutcome, v.anonUserRest.MaxAnnualOutcome)
			return &results.ExceededLimitError{Description: description}, nil
		}
	}

	return nil, nil
}

func (v *OutgoingLimitsValidator) getDailyOutcome() (int64, error) {
	if v.dailyOutcome != nil {
		return *v.dailyOutcome, nil
	}
	v.dailyOutcome = new(int64)
	stats, err := v.getAccountStats()
	if err != nil {
		return 0, err
	}

	*v.dailyOutcome = helpers.SumAccountStats(
		stats,
		func(stats *history.AccountStatistics) int64 { return stats.DailyOutcome },
		xdr.AccountTypeAccountAnonymousUser,
		xdr.AccountTypeAccountRegisteredUser,
		xdr.AccountTypeAccountSettlementAgent,
	)
	return *v.dailyOutcome, nil
}

func (v *OutgoingLimitsValidator) getMonthlyOutcome() (int64, error) {
	if v.monthlyOutcome != nil {
		return *v.monthlyOutcome, nil
	}
	v.monthlyOutcome = new(int64)
	stats, err := v.getAccountStats()
	if err != nil {
		return 0, err
	}

	*v.monthlyOutcome = helpers.SumAccountStats(
		stats,
		func(stats *history.AccountStatistics) int64 { return stats.MonthlyOutcome },
		xdr.AccountTypeAccountAnonymousUser,
		xdr.AccountTypeAccountRegisteredUser,
		xdr.AccountTypeAccountSettlementAgent,
	)
	return *v.monthlyOutcome, nil
}
