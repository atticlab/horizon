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

func TestOutgoingLimits(t *testing.T) {
	log.DefaultLogger.Entry.Logger.Level = log.DebugLevel

	sourceKey, err := keypair.Random()
	assert.Nil(t, err)
	destKey, err := keypair.Random()
	assert.Nil(t, err)
	source := &core.Account{
		Accountid:   sourceKey.Address(),
		AccountType: xdr.AccountTypeAccountAnonymousUser,
	}
	dest := &core.Account{
		Accountid:   destKey.Address(),
		AccountType: xdr.AccountTypeAccountAnonymousUser,
	}
	opAmount := int64(amount.One * 100)
	opAsset := history.Asset{
		Code:        "UAH",
		IsAnonymous: false,
	}
	sourceLimits := history.AccountLimits{
		Account:         source.Accountid,
		AssetCode:       opAsset.Code,
		MaxOperationOut: -1,
		DailyMaxOut:     -1,
		MonthlyMaxOut:   -1,
		MaxOperationIn:  -1,
		DailyMaxIn:      -1,
		MonthlyMaxIn:    -1,
	}
	Convey("Outgoing limits test:", t, func() {
		Convey("No limits for source & asset is not anonymous", func() {
			histMock := history.QMock{}
			histMock.On("GetAccountLimits", source.Accountid, opAsset.Code).Return(nil, sql.ErrNoRows)
			v := NewOutgoingLimitsValidator(source, dest, opAmount, opAsset, &histMock, config.AnonymousUserRestrictions{})
			result, err := v.VerifyLimits()
			So(err, ShouldBeNil)
			So(result, ShouldBeNil)
		})
		Convey("All limits are empty for source & asset is not anonymous", func() {
			histMock := history.QMock{}
			limits := sourceLimits
			histMock.On("GetAccountLimits", source.Accountid, opAsset.Code).Return(limits, nil)
			v := NewOutgoingLimitsValidator(source, dest, opAmount, opAsset, &histMock, config.AnonymousUserRestrictions{})
			result, err := v.VerifyLimits()
			So(err, ShouldBeNil)
			So(result, ShouldBeNil)
		})
		Convey("Asset is not anonymous, exceeds op amount", func() {
			limits := sourceLimits
			limits.MaxOperationOut = opAmount - 1
			histMock := history.QMock{}
			histMock.On("GetAccountLimits", source.Accountid, opAsset.Code).Return(limits, nil)
			v := NewOutgoingLimitsValidator(source, dest, opAmount, opAsset, &histMock, config.AnonymousUserRestrictions{})
			result, err := v.VerifyLimits()
			So(err, ShouldBeNil)
			assert.Equal(t, &results.ExceededLimitError{Description: fmt.Sprintf(
				"Maximal operation amount for account (%s) exceeded: %s of %s %s",
				source.Accountid,
				amount.String(xdr.Int64(opAmount)),
				amount.String(xdr.Int64(limits.MaxOperationOut)),
				opAsset.Code,
			)}, result)
		})
		Convey("Asset is not anonymous, exceeds daily limit with empty stats", func() {
			limits := sourceLimits
			limits.DailyMaxOut = opAmount - 1
			histMock := history.QMock{}
			histMock.On("GetAccountLimits", source.Accountid, opAsset.Code).Return(limits, nil)
			histMock.On("GetStatisticsByAccountAndAsset", source.Accountid, opAsset.Code).Return(nil, sql.ErrNoRows)
			v := NewOutgoingLimitsValidator(source, dest, opAmount, opAsset, &histMock, config.AnonymousUserRestrictions{})
			result, err := v.VerifyLimits()
			So(err, ShouldBeNil)
			assert.Equal(t, &results.ExceededLimitError{Description: fmt.Sprintf("Daily outgoing payments limit for account exceeded: %s + %s out of %s %s.",
				amount.String(xdr.Int64(xdr.Int64(0))),
				amount.String(xdr.Int64(opAmount)),
				amount.String(xdr.Int64(limits.DailyMaxOut)),
				opAsset.Code,
			)}, result)
		})
		Convey("Asset is not anonymous, exceeds daily limit with stats", func() {
			limits := sourceLimits
			limits.DailyMaxOut = 2*opAmount - 1
			histMock := history.QMock{}
			histMock.On("GetAccountLimits", source.Accountid, opAsset.Code).Return(limits, nil)
			stats := map[xdr.AccountType]history.AccountStatistics{
				xdr.AccountTypeAccountAnonymousUser: history.AccountStatistics{
					Account:          source.Accountid,
					AssetCode:        opAsset.Code,
					CounterpartyType: int16(xdr.AccountTypeAccountSettlementAgent),
					DailyOutcome:     opAmount,
				},
			}
			histMock.On("GetStatisticsByAccountAndAsset", source.Accountid, opAsset.Code).Return(stats, nil)
			v := NewOutgoingLimitsValidator(source, dest, opAmount, opAsset, &histMock, config.AnonymousUserRestrictions{})
			result, err := v.VerifyLimits()
			So(err, ShouldBeNil)
			assert.Equal(t, &results.ExceededLimitError{Description: fmt.Sprintf("Daily outgoing payments limit for account exceeded: %s + %s out of %s %s.",
				amount.String(xdr.Int64(xdr.Int64(opAmount))),
				amount.String(xdr.Int64(opAmount)),
				amount.String(xdr.Int64(limits.DailyMaxOut)),
				opAsset.Code,
			)}, result)
		})
		Convey("Asset is not anonymous, exceeds monthly limit with empty stats", func() {
			limits := sourceLimits
			limits.MonthlyMaxOut = 2*opAmount - 1
			histMock := history.QMock{}
			histMock.On("GetAccountLimits", source.Accountid, opAsset.Code).Return(limits, nil)
			stats := map[xdr.AccountType]history.AccountStatistics{
				xdr.AccountTypeAccountAnonymousUser: history.AccountStatistics{
					Account:          source.Accountid,
					AssetCode:        opAsset.Code,
					CounterpartyType: int16(xdr.AccountTypeAccountSettlementAgent),
					MonthlyOutcome:     opAmount,
				},
			}
			histMock.On("GetStatisticsByAccountAndAsset", source.Accountid, opAsset.Code).Return(stats, nil)
			v := NewOutgoingLimitsValidator(source, dest, opAmount, opAsset, &histMock, config.AnonymousUserRestrictions{})
			result, err := v.VerifyLimits()
			So(err, ShouldBeNil)
			assert.Equal(t, &results.ExceededLimitError{Description: fmt.Sprintf("Monthly outgoing payments limit for account exceeded: %s + %s out of %s %s.",
				amount.String(xdr.Int64(xdr.Int64(opAmount))),
				amount.String(xdr.Int64(opAmount)),
				amount.String(xdr.Int64(limits.MonthlyMaxOut)),
				opAsset.Code,
			)}, result)
		})
		stats := map[xdr.AccountType]history.AccountStatistics{
			xdr.AccountTypeAccountAnonymousUser: history.AccountStatistics{
				Account:          source.Accountid,
				AssetCode:        opAsset.Code,
				CounterpartyType: int16(xdr.AccountTypeAccountSettlementAgent),
				DailyOutcome:     opAmount,
				MonthlyOutcome: opAmount,
				AnnualOutcome: opAmount,
			},
		}
		Convey("Asset is anonymous exceeds dayli limit, with no account limits", func() {
			limits := config.AnonymousUserRestrictions{
				MaxDailyOutcome: 2*opAmount - 1,
			}
			opAsset.IsAnonymous = true
			histMock := history.QMock{}
			histMock.On("GetAccountLimits", source.Accountid, opAsset.Code).Return(nil, sql.ErrNoRows)
			histMock.On("GetStatisticsByAccountAndAsset", source.Accountid, opAsset.Code).Return(stats, nil)
			v := NewOutgoingLimitsValidator(source, dest, opAmount, opAsset, &histMock, limits)
			result, err := v.VerifyLimits()
			So(err, ShouldBeNil)
			assert.Equal(t, &results.ExceededLimitError{Description: fmt.Sprintf("Daily outgoing payments limit for anonymous account exceeded: %s + %s out of %s %s.",
				amount.String(xdr.Int64(xdr.Int64(opAmount))),
				amount.String(xdr.Int64(opAmount)),
				amount.String(xdr.Int64(limits.MaxDailyOutcome)),
				opAsset.Code,
			)}, result)
		})
		Convey("Asset is anonymous exceeds monthly limit, with no account limits", func() {
			limits := config.AnonymousUserRestrictions{
				MaxDailyOutcome: 2*opAmount,
				MaxMonthlyOutcome: 2*opAmount - 1,
			}
			opAsset.IsAnonymous = true
			histMock := history.QMock{}
			histMock.On("GetAccountLimits", source.Accountid, opAsset.Code).Return(nil, sql.ErrNoRows)
			histMock.On("GetStatisticsByAccountAndAsset", source.Accountid, opAsset.Code).Return(stats, nil)
			v := NewOutgoingLimitsValidator(source, dest, opAmount, opAsset, &histMock, limits)
			result, err := v.VerifyLimits()
			So(err, ShouldBeNil)
			assert.Equal(t, &results.ExceededLimitError{Description: fmt.Sprintf("Monthly outgoing payments limit for anonymous account exceeded: %s + %s out of %s %s.",
				amount.String(xdr.Int64(xdr.Int64(opAmount))),
				amount.String(xdr.Int64(opAmount)),
				amount.String(xdr.Int64(limits.MaxMonthlyOutcome)),
				opAsset.Code,
			)}, result)
		})
		Convey("Asset is anonymous exceeds annual limit, with no account limits", func() {
			limits := config.AnonymousUserRestrictions{
				MaxDailyOutcome: 2*opAmount,
				MaxMonthlyOutcome: 2*opAmount,
				MaxAnnualOutcome: 2*opAmount - 1,
			}
			opAsset.IsAnonymous = true
			histMock := history.QMock{}
			histMock.On("GetAccountLimits", source.Accountid, opAsset.Code).Return(nil, sql.ErrNoRows)
			histMock.On("GetStatisticsByAccountAndAsset", source.Accountid, opAsset.Code).Return(stats, nil)
			v := NewOutgoingLimitsValidator(source, dest, opAmount, opAsset, &histMock, limits)
			result, err := v.VerifyLimits()
			So(err, ShouldBeNil)
			assert.Equal(t, &results.ExceededLimitError{Description: fmt.Sprintf("Annual outgoing payments limit for anonymous account exceeded: %s + %s out of %s %s.",
				amount.String(xdr.Int64(xdr.Int64(opAmount))),
				amount.String(xdr.Int64(opAmount)),
				amount.String(xdr.Int64(limits.MaxAnnualOutcome)),
				opAsset.Code,
			)}, result)
		})
		Convey("Asset is anonymous to SettlementAgent, with no account limits", func() {
			limits := config.AnonymousUserRestrictions{
				MaxDailyOutcome: 2*opAmount,
				MaxMonthlyOutcome: 2*opAmount,
				MaxAnnualOutcome: 2*opAmount - 1,
			}
			opAsset.IsAnonymous = true
			histMock := history.QMock{}
			histMock.On("GetAccountLimits", source.Accountid, opAsset.Code).Return(nil, sql.ErrNoRows)
			histMock.On("GetStatisticsByAccountAndAsset", source.Accountid, opAsset.Code).Return(stats, nil)
			dest.AccountType = xdr.AccountTypeAccountSettlementAgent
			v := NewOutgoingLimitsValidator(source, dest, opAmount, opAsset, &histMock, limits)
			result, err := v.VerifyLimits()
			So(err, ShouldBeNil)
			So(result, ShouldBeNil)
		})
		Convey("Asset is anonymous to merchant", func() {
			limits := config.AnonymousUserRestrictions{
				MaxDailyOutcome: 2*opAmount - 1,
				MaxMonthlyOutcome: 2*opAmount - 1,
				MaxAnnualOutcome: 2*opAmount,
			}
			opAsset.IsAnonymous = true
			histMock := history.QMock{}
			histMock.On("GetAccountLimits", source.Accountid, opAsset.Code).Return(nil, sql.ErrNoRows)
			histMock.On("GetStatisticsByAccountAndAsset", source.Accountid, opAsset.Code).Return(stats, nil)
			dest.AccountType = xdr.AccountTypeAccountMerchant
			v := NewOutgoingLimitsValidator(source, dest, opAmount, opAsset, &histMock, limits)
			result, err := v.VerifyLimits()
			So(err, ShouldBeNil)
			So(result, ShouldBeNil)
		})


	})
}
