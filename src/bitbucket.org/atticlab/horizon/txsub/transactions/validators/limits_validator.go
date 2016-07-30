package validators

import (
	"bitbucket.org/atticlab/go-smart-base/amount"
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/config"
	"bitbucket.org/atticlab/horizon/db2/core"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/log"
	"database/sql"
	"fmt"
	"time"
)

type PaymentType string

const (
	PaymentTypeOutgoing PaymentType = "outgoing"
	PaymentTypeIncoming PaymentType = "incoming"
)

type limitsValidator struct {
	account      *core.Account
	counterparty *core.Account
	opAmount     int64
	opAsset      history.Asset
	historyQ     history.QInterface
	anonUserRest config.AnonymousUserRestrictions
	accountStats map[xdr.AccountType]history.AccountStatistics
	paymentType  PaymentType
	log          *log.Entry
	now          time.Time
}

func newLimitsValidator(paymentType PaymentType, sender, destination *core.Account, opAmount int64, opAsset history.Asset, historyQ history.QInterface, anonUserRestr config.AnonymousUserRestrictions, now time.Time) *limitsValidator {
	return &limitsValidator{
		account:      sender,
		counterparty: destination,
		opAsset:      opAsset,
		opAmount:     opAmount,
		historyQ:     historyQ,
		anonUserRest: anonUserRestr,
		paymentType:  paymentType,
		log:          log.WithField("service", "limits_validator"),
		now:          now,
	}
}

func (v *limitsValidator) getAccountStats() (map[xdr.AccountType]history.AccountStatistics, error) {
	if v.accountStats != nil {
		return v.accountStats, nil
	}
	v.accountStats = make(map[xdr.AccountType]history.AccountStatistics)
	err := v.historyQ.GetStatisticsByAccountAndAsset(v.accountStats, v.account.Accountid, v.opAsset.Code, v.now)
	if err != nil {
		if err != sql.ErrNoRows {
			return nil, err
		}
	}
	return v.accountStats, nil
}

func (v *limitsValidator) limitExceededDescription(periodName string, isAnonymous bool, outcome, limit int64) string {
	anonymous := ""
	if isAnonymous {
		anonymous = "anonymous "
	}
	return fmt.Sprintf("%s %s payments limit for %saccount exceeded: %s + %s out of %s %s.",
		periodName,
		v.paymentType,
		anonymous,
		amount.String(xdr.Int64(xdr.Int64(outcome))),
		amount.String(xdr.Int64(v.opAmount)),
		amount.String(xdr.Int64(limit)),
		v.opAsset.Code,
	)
}
