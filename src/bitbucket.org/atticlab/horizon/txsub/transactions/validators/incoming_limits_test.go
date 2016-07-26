package validators

import (
	"testing"

	"bitbucket.org/atticlab/go-smart-base/amount"
	"bitbucket.org/atticlab/go-smart-base/keypair"
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/config"
	"bitbucket.org/atticlab/horizon/db2/core"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/log"
	"bitbucket.org/atticlab/horizon/txsub/results"
	"database/sql"
	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
)

func TestIncomingLimits(t *testing.T) {
	log.DefaultLogger.Entry.Logger.Level = log.DebugLevel

	sourceKey, err := keypair.Random()
	assert.Nil(t, err)
	destKey, err := keypair.Random()
	assert.Nil(t, err)
	account := &core.Account{
		Accountid:   sourceKey.Address(),
		AccountType: xdr.AccountTypeAccountAnonymousUser,
	}
	counterparty := &core.Account{
		Accountid:   destKey.Address(),
		AccountType: xdr.AccountTypeAccountAnonymousUser,
	}
	opAmount := int64(amount.One * 100)
	opAsset := history.Asset{
		Code:        "UAH",
		IsAnonymous: false,
	}
	accountLimits := history.AccountLimits{
		Account:         account.Accountid,
		AssetCode:       opAsset.Code,
		MaxOperationOut: -1,
		DailyMaxOut:     -1,
		MonthlyMaxOut:   -1,
		MaxOperationIn:  -1,
		DailyMaxIn:      -1,
		MonthlyMaxIn:    -1,
	}

	accountTrustLine := core.Trustline{
		Accountid: account.Accountid,
		Balance: 0,
	}
	Convey("Incoming limits test:", t, func() {
		Convey("No limits for account & asset is not anonymous", func() {
			histMock := history.QMock{}
			histMock.On("GetAccountLimits", account.Accountid, opAsset.Code).Return(nil, sql.ErrNoRows)
			v := NewIncomingLimitsValidator(account, counterparty, accountTrustLine, opAmount, opAsset, &histMock, config.AnonymousUserRestrictions{})
			result, err := v.VerifyLimits()
			So(err, ShouldBeNil)
			So(result, ShouldBeNil)
		})
		Convey("All limits are empty for account & asset is not anonymous", func() {
			histMock := history.QMock{}
			limits := accountLimits
			histMock.On("GetAccountLimits", account.Accountid, opAsset.Code).Return(limits, nil)
			v := NewIncomingLimitsValidator(account, counterparty, accountTrustLine, opAmount, opAsset, &histMock, config.AnonymousUserRestrictions{})
			result, err := v.VerifyLimits()
			So(err, ShouldBeNil)
			So(result, ShouldBeNil)
		})
		Convey("Asset is not anonymous, exceeds op amount", func() {
			limits := accountLimits
			limits.MaxOperationIn = opAmount - 1
			histMock := history.QMock{}
			histMock.On("GetAccountLimits", account.Accountid, opAsset.Code).Return(limits, nil)
			v := NewIncomingLimitsValidator(account, counterparty, accountTrustLine, opAmount, opAsset, &histMock, config.AnonymousUserRestrictions{})
			result, err := v.VerifyLimits()
			So(err, ShouldBeNil)
			assert.Equal(t, &results.ExceededLimitError{Description: fmt.Sprintf(
				"Maximal operation amount for account (%s) exceeded: %s of %s %s",
				account.Accountid,
				amount.String(xdr.Int64(opAmount)),
				amount.String(xdr.Int64(limits.MaxOperationIn)),
				opAsset.Code,
			)}, result)
		})
		Convey("Asset is not anonymous, exceeds daily limit with empty stats", func() {
			limits := accountLimits
			limits.DailyMaxIn = opAmount - 1
			histMock := history.QMock{}
			histMock.On("GetAccountLimits", account.Accountid, opAsset.Code).Return(limits, nil)
			histMock.On("GetStatisticsByAccountAndAsset", account.Accountid, opAsset.Code).Return(nil, sql.ErrNoRows)
			v := NewIncomingLimitsValidator(account, counterparty, accountTrustLine, opAmount, opAsset, &histMock, config.AnonymousUserRestrictions{})
			result, err := v.VerifyLimits()
			So(err, ShouldBeNil)
			assert.Equal(t, &results.ExceededLimitError{Description: fmt.Sprintf("Daily incoming payments limit for account exceeded: %s + %s out of %s %s.",
				amount.String(xdr.Int64(xdr.Int64(0))),
				amount.String(xdr.Int64(opAmount)),
				amount.String(xdr.Int64(limits.DailyMaxIn)),
				opAsset.Code,
			)}, result)
		})
		Convey("Asset is not anonymous, exceeds daily limit with stats", func() {
			limits := accountLimits
			limits.DailyMaxIn = 2*opAmount - 1
			histMock := history.QMock{}
			histMock.On("GetAccountLimits", account.Accountid, opAsset.Code).Return(limits, nil)
			stats := map[xdr.AccountType]history.AccountStatistics{
				xdr.AccountTypeAccountAnonymousUser: history.AccountStatistics{
					Account:          account.Accountid,
					AssetCode:        opAsset.Code,
					CounterpartyType: int16(xdr.AccountTypeAccountSettlementAgent),
					DailyIncome:     opAmount,
				},
			}
			histMock.On("GetStatisticsByAccountAndAsset", account.Accountid, opAsset.Code).Return(stats, nil)
			v := NewIncomingLimitsValidator(account, counterparty, accountTrustLine, opAmount, opAsset, &histMock, config.AnonymousUserRestrictions{})
			result, err := v.VerifyLimits()
			So(err, ShouldBeNil)
			assert.Equal(t, &results.ExceededLimitError{Description: fmt.Sprintf("Daily incoming payments limit for account exceeded: %s + %s out of %s %s.",
				amount.String(xdr.Int64(xdr.Int64(opAmount))),
				amount.String(xdr.Int64(opAmount)),
				amount.String(xdr.Int64(limits.DailyMaxIn)),
				opAsset.Code,
			)}, result)
		})
		Convey("Asset is not anonymous, exceeds monthly limit with empty stats", func() {
			limits := accountLimits
			limits.MonthlyMaxIn = 2*opAmount - 1
			histMock := history.QMock{}
			histMock.On("GetAccountLimits", account.Accountid, opAsset.Code).Return(limits, nil)
			stats := map[xdr.AccountType]history.AccountStatistics{
				xdr.AccountTypeAccountAnonymousUser: history.AccountStatistics{
					Account:          account.Accountid,
					AssetCode:        opAsset.Code,
					CounterpartyType: int16(xdr.AccountTypeAccountSettlementAgent),
					MonthlyIncome:     opAmount,
				},
			}
			histMock.On("GetStatisticsByAccountAndAsset", account.Accountid, opAsset.Code).Return(stats, nil)
			v := NewIncomingLimitsValidator(account, counterparty, accountTrustLine, opAmount, opAsset, &histMock, config.AnonymousUserRestrictions{})
			result, err := v.VerifyLimits()
			So(err, ShouldBeNil)
			assert.Equal(t, &results.ExceededLimitError{Description: fmt.Sprintf("Monthly incoming payments limit for account exceeded: %s + %s out of %s %s.",
				amount.String(xdr.Int64(xdr.Int64(opAmount))),
				amount.String(xdr.Int64(opAmount)),
				amount.String(xdr.Int64(limits.MonthlyMaxIn)),
				opAsset.Code,
			)}, result)
		})
		stats := map[xdr.AccountType]history.AccountStatistics{
			xdr.AccountTypeAccountAnonymousUser: history.AccountStatistics{
				Account:          account.Accountid,
				AssetCode:        opAsset.Code,
				CounterpartyType: int16(xdr.AccountTypeAccountSettlementAgent),
				DailyIncome:     opAmount,
				MonthlyIncome: opAmount,
				AnnualIncome: opAmount,
			},
		}
		Convey("Asset is anonymous exceeds max balance", func() {
			limits := config.AnonymousUserRestrictions{
				MaxBalance: 2*opAmount - 1,
			}
			accountTrustLine.Balance = xdr.Int64(opAmount)
			opAsset.IsAnonymous = true
			histMock := history.QMock{}
			histMock.On("GetAccountLimits", account.Accountid, opAsset.Code).Return(nil, sql.ErrNoRows)
			histMock.On("GetStatisticsByAccountAndAsset", account.Accountid, opAsset.Code).Return(stats, nil)
			v := NewIncomingLimitsValidator(account, counterparty, accountTrustLine, opAmount, opAsset, &histMock, limits)
			result, err := v.VerifyLimits()
			So(err, ShouldBeNil)
			assert.Equal(t, &results.ExceededLimitError{Description: fmt.Sprintf(
				"User's max balance exceeded: %s + %s out of %s UAH.",
				amount.String(accountTrustLine.Balance),
				amount.String(xdr.Int64(opAmount)),
				amount.String(xdr.Int64(limits.MaxBalance)),
			)}, result)
		})
		Convey("Asset is anonymous exceeds max balance, but is not user", func() {
			limits := config.AnonymousUserRestrictions{
				MaxBalance: 2*opAmount - 1,
			}
			accountTrustLine.Balance = xdr.Int64(opAmount)
			opAsset.IsAnonymous = true
			histMock := history.QMock{}
			histMock.On("GetAccountLimits", account.Accountid, opAsset.Code).Return(nil, sql.ErrNoRows)
			histMock.On("GetStatisticsByAccountAndAsset", account.Accountid, opAsset.Code).Return(stats, nil)
			account.AccountType = xdr.AccountTypeAccountMerchant
			v := NewIncomingLimitsValidator(account, counterparty, accountTrustLine, opAmount, opAsset, &histMock, limits)
			result, err := v.VerifyLimits()
			So(err, ShouldBeNil)
			So(result, ShouldBeNil)
		})


	})
}
