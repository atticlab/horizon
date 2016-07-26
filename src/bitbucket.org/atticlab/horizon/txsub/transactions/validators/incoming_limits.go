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

type IncomingLimitsValidatorInterface interface {
	VerifyLimits() (*results.ExceededLimitError, error)
}

type IncomingLimitsValidator struct {
	limitsValidator
	accountTrustLine core.Trustline
	dailyIncome      *int64
	monthlyIncome    *int64
}

func NewIncomingLimitsValidator(account, counterparty *core.Account, accountTrustLine core.Trustline, opAmount int64, opAsset history.Asset, historyQ history.QInterface, anonUserRestr config.AnonymousUserRestrictions) *IncomingLimitsValidator {
	limitsValidator := newLimitsValidator(PaymentTypeIncoming, account, counterparty, opAmount, opAsset, historyQ, anonUserRestr)
	result := &IncomingLimitsValidator{
		limitsValidator:  *limitsValidator,
		accountTrustLine: accountTrustLine,
	}
	result.log = log.WithField("service", "incoming_limits_validator")
	return result
}

// VerifyLimits checks incoming limits
func (v *IncomingLimitsValidator) VerifyLimits() (*results.ExceededLimitError, error) {
	// check account's limits
	result, err := v.verifyReceiverAccountLimits()
	if result != nil || err != nil {
		return result, err
	}

	return v.verifyAnonymousAssetLimits()
}

func (v *IncomingLimitsValidator) verifyReceiverAccountLimits() (*results.ExceededLimitError, error) {
	var limits history.AccountLimits
	err := v.historyQ.GetAccountLimits(&limits, v.account.Accountid, v.opAsset.Code)
	if err != nil {
		// no limits to check for destination
		if err == sql.ErrNoRows {
			v.log.Debug("No limits found")
			return nil, nil
		}
		return nil, err
	}

	v.log.WithField("limits", limits).Debug("Checking limits")
	if limits.MaxOperationIn >= 0 && v.opAmount > limits.MaxOperationIn {
		description := fmt.Sprintf(
			"Maximal operation amount for account (%s) exceeded: %s of %s %s",
			v.account.Accountid,
			amount.String(xdr.Int64(v.opAmount)),
			amount.String(xdr.Int64(limits.MaxOperationIn)),
			v.opAsset.Code,
		)
		return &results.ExceededLimitError{Description: description}, nil
	}

	if limits.DailyMaxIn >= 0 {
		dailyIncome, err := v.getDailyIncome()
		if err != nil {
			return nil, err
		}
		v.log.WithFields(log.F{
			"newIncome": dailyIncome + v.opAmount,
			"limit":     limits.DailyMaxIn,
		}).Debug("Checking daily outcome for limits")
		if dailyIncome+v.opAmount > limits.DailyMaxIn {
			description := v.limitExceededDescription("Daily", false, dailyIncome, limits.DailyMaxIn)
			return &results.ExceededLimitError{Description: description}, nil
		}
	}

	if limits.MonthlyMaxIn >= 0 {
		monthlyIncome, err := v.getMonthlyIncome()
		if err != nil {
			return nil, err
		}
		v.log.WithFields(log.F{
			"newIncome": monthlyIncome + v.opAmount,
			"limit":     limits.MonthlyMaxIn,
		}).Debug("Checking daily income for limits")
		if monthlyIncome+v.opAmount > limits.MonthlyMaxIn {
			description := v.limitExceededDescription("Monthly", false, monthlyIncome, limits.MonthlyMaxIn)
			return &results.ExceededLimitError{Description: description}, nil
		}
	}
	return nil, nil
}

// VerifyLimitsForReceiver checks limits  and restrictions for receiver
func (v *IncomingLimitsValidator) verifyAnonymousAssetLimits() (*results.ExceededLimitError, error) {
	if !v.opAsset.IsAnonymous || !helpers.IsUser(v.account.AccountType) {
		// Nothing to be checked
		return nil, nil
	}

	if int64(v.accountTrustLine.Balance)+v.opAmount > v.anonUserRest.MaxBalance {
		description := fmt.Sprintf(
			"User's max balance exceeded: %s + %s out of %s UAH.",
			amount.String(v.accountTrustLine.Balance),
			amount.String(xdr.Int64(v.opAmount)),
			amount.String(xdr.Int64(v.anonUserRest.MaxBalance)),
		)
		return &results.ExceededLimitError{Description: description}, nil
	}
	return nil, nil
}

func (v *IncomingLimitsValidator) getDailyIncome() (int64, error) {
	if v.dailyIncome != nil {
		return *v.dailyIncome, nil
	}
	v.dailyIncome = new(int64)
	stats, err := v.getAccountStats()
	if err != nil {
		return 0, err
	}

	*v.dailyIncome = helpers.SumAccountStats(
		stats,
		func(stats *history.AccountStatistics) int64 { return stats.DailyIncome },
		xdr.AccountTypeAccountAnonymousUser,
		xdr.AccountTypeAccountRegisteredUser,
		xdr.AccountTypeAccountSettlementAgent,
	)
	return *v.dailyIncome, nil
}

func (v *IncomingLimitsValidator) getMonthlyIncome() (int64, error) {
	if v.monthlyIncome != nil {
		return *v.monthlyIncome, nil
	}
	v.monthlyIncome = new(int64)
	stats, err := v.getAccountStats()
	if err != nil {
		return 0, err
	}

	*v.monthlyIncome = helpers.SumAccountStats(
		stats,
		func(stats *history.AccountStatistics) int64 { return stats.MonthlyIncome },
		xdr.AccountTypeAccountAnonymousUser,
		xdr.AccountTypeAccountRegisteredUser,
		xdr.AccountTypeAccountSettlementAgent,
	)
	return *v.monthlyIncome, nil
}
