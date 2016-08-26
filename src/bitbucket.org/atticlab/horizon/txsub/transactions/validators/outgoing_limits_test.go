package validators

import (
	"testing"

	"bitbucket.org/atticlab/go-smart-base/amount"
	"bitbucket.org/atticlab/go-smart-base/keypair"
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/config"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/log"
	"bitbucket.org/atticlab/horizon/txsub/results"
	"bitbucket.org/atticlab/horizon/txsub/transactions/statistics"
	"database/sql"
	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"time"
)

func TestOutgoingLimits(t *testing.T) {
	log.DefaultLogger.Entry.Logger.Level = log.DebugLevel

	sourceKey, err := keypair.Random()
	assert.Nil(t, err)
	destKey, err := keypair.Random()
	assert.Nil(t, err)
	source := &history.Account{
		Address:     sourceKey.Address(),
		AccountType: xdr.AccountTypeAccountAnonymousUser,
	}
	dest := &history.Account{
		Address:     destKey.Address(),
		AccountType: xdr.AccountTypeAccountAnonymousUser,
	}
	opAmount := int64(amount.One * 100)
	opAsset := history.Asset{
		Code:        "UAH",
		IsAnonymous: false,
	}
	sourceLimits := history.AccountLimits{
		Account:         source.Address,
		AssetCode:       opAsset.Code,
		MaxOperationOut: -1,
		DailyMaxOut:     -1,
		MonthlyMaxOut:   -1,
		MaxOperationIn:  -1,
		DailyMaxIn:      -1,
		MonthlyMaxIn:    -1,
	}

	now := time.Now()

	opData := statistics.NewOperationData(source, 0, "random_tx_hash")
	paymentData := statistics.NewPaymentData(dest, opAsset, opAmount, opData)
	direction := statistics.PaymentDirectionOutgoing

	statsManager := statistics.ManagerMock{}

	Convey("Outgoing limits test:", t, func() {
		Convey("No limits for source & asset is not anonymous", func() {
			histMock := history.QMock{}
			histMock.On("GetAccountLimits", paymentData.GetAccount(direction).Address, opAsset.Code).Return(nil, sql.ErrNoRows)
			v := NewOutgoingLimitsValidator(&paymentData, &statsManager, &histMock, config.AnonymousUserRestrictions{}, now)
			result, err := v.VerifyLimits()
			So(err, ShouldBeNil)
			So(result, ShouldBeNil)
		})
		Convey("All limits are empty for source & asset is not anonymous", func() {
			histMock := history.QMock{}
			limits := sourceLimits
			histMock.On("GetAccountLimits", paymentData.GetAccount(direction).Address, opAsset.Code).Return(limits, nil)
			v := NewOutgoingLimitsValidator(&paymentData, &statsManager, &histMock, config.AnonymousUserRestrictions{}, now)
			result, err := v.VerifyLimits()
			So(err, ShouldBeNil)
			So(result, ShouldBeNil)
		})
		Convey("Asset is not anonymous, exceeds op amount", func() {
			limits := sourceLimits
			limits.MaxOperationOut = opAmount - 1
			histMock := history.QMock{}
			histMock.On("GetAccountLimits", paymentData.GetAccount(direction).Address, opAsset.Code).Return(limits, nil)
			v := NewOutgoingLimitsValidator(&paymentData, &statsManager, &histMock, config.AnonymousUserRestrictions{}, now)
			result, err := v.VerifyLimits()
			So(err, ShouldBeNil)
			assert.Equal(t, &results.ExceededLimitError{Description: fmt.Sprintf(
				"Maximal operation amount for account (%s) exceeded: %s of %s %s",
				paymentData.GetAccount(direction).Address,
				amount.String(xdr.Int64(opAmount)),
				amount.String(xdr.Int64(limits.MaxOperationOut)),
				opAsset.Code,
			)}, result)
		})
		Convey("Asset is not anonymous, exceeds daily limit with empty stats", func() {
			limits := sourceLimits
			limits.DailyMaxOut = opAmount - 1
			histMock := history.QMock{}
			histMock.On("GetAccountLimits", paymentData.GetAccount(direction).Address, opAsset.Code).Return(limits, nil)
			statsManager.On("UpdateGet", &paymentData, direction, now).Return(map[xdr.AccountType]history.AccountStatistics{
				paymentData.GetCounterparty(direction).AccountType: history.AccountStatistics{
					DailyOutcome: opAmount,
				},
			}, nil).Once()
			v := NewOutgoingLimitsValidator(&paymentData, &statsManager, &histMock, config.AnonymousUserRestrictions{}, now)
			result, err := v.VerifyLimits()
			So(err, ShouldBeNil)
			assert.Equal(t, &results.ExceededLimitError{Description: fmt.Sprintf("Daily outgoing payments limit for account exceeded: %s out of %s %s.",
				amount.String(xdr.Int64(opAmount)),
				amount.String(xdr.Int64(limits.DailyMaxOut)),
				opAsset.Code,
			)}, result)
		})
		Convey("Asset is not anonymous, exceeds daily limit with stats", func() {
			limits := sourceLimits
			limits.DailyMaxOut = 2*opAmount - 1
			histMock := history.QMock{}
			histMock.On("GetAccountLimits", paymentData.GetAccount(direction).Address, opAsset.Code).Return(limits, nil)
			stats := map[xdr.AccountType]history.AccountStatistics{
				xdr.AccountTypeAccountSettlementAgent: history.AccountStatistics{
					Account:          paymentData.GetAccount(direction).Address,
					AssetCode:        opAsset.Code,
					CounterpartyType: int16(xdr.AccountTypeAccountSettlementAgent),
					DailyOutcome:     opAmount + opAmount,
				},
			}
			statsManager.On("UpdateGet", &paymentData, direction, now).Return(stats, nil).Once()
			v := NewOutgoingLimitsValidator(&paymentData, &statsManager, &histMock, config.AnonymousUserRestrictions{}, now)
			result, err := v.VerifyLimits()
			So(err, ShouldBeNil)
			assert.Equal(t, &results.ExceededLimitError{Description: fmt.Sprintf("Daily outgoing payments limit for account exceeded: %s out of %s %s.",
				amount.String(xdr.Int64(opAmount+opAmount)),
				amount.String(xdr.Int64(limits.DailyMaxOut)),
				opAsset.Code,
			)}, result)
		})
		Convey("Asset is not anonymous, exceeds monthly limit with empty stats", func() {
			limits := sourceLimits
			limits.MonthlyMaxOut = 2*opAmount - 1
			histMock := history.QMock{}
			histMock.On("GetAccountLimits", paymentData.GetAccount(direction).Address, opAsset.Code).Return(limits, nil)
			stats := map[xdr.AccountType]history.AccountStatistics{
				xdr.AccountTypeAccountAnonymousUser: history.AccountStatistics{
					Account:          paymentData.GetAccount(direction).Address,
					AssetCode:        opAsset.Code,
					CounterpartyType: int16(xdr.AccountTypeAccountSettlementAgent),
					MonthlyOutcome:   opAmount + opAmount,
				},
			}
			statsManager.On("UpdateGet", &paymentData, direction, now).Return(stats, nil).Once()
			v := NewOutgoingLimitsValidator(&paymentData, &statsManager, &histMock, config.AnonymousUserRestrictions{}, now)
			result, err := v.VerifyLimits()
			So(err, ShouldBeNil)
			assert.Equal(t, &results.ExceededLimitError{Description: fmt.Sprintf("Monthly outgoing payments limit for account exceeded: %s out of %s %s.",
				amount.String(xdr.Int64(opAmount+opAmount)),
				amount.String(xdr.Int64(limits.MonthlyMaxOut)),
				opAsset.Code,
			)}, result)
		})
		stats := map[xdr.AccountType]history.AccountStatistics{
			xdr.AccountTypeAccountAnonymousUser: history.AccountStatistics{
				Account:          paymentData.GetAccount(direction).Address,
				AssetCode:        opAsset.Code,
				CounterpartyType: int16(xdr.AccountTypeAccountSettlementAgent),
				DailyOutcome:     opAmount + opAmount,
				MonthlyOutcome:   opAmount + opAmount,
				AnnualOutcome:    opAmount + opAmount,
			},
		}
		Convey("Asset is anonymous exceeds dayli limit, with no account limits", func() {
			limits := config.AnonymousUserRestrictions{
				MaxDailyOutcome: 2*opAmount - 1,
			}
			paymentData.Asset.IsAnonymous = true
			histMock := history.QMock{}
			histMock.On("GetAccountLimits", paymentData.GetAccount(direction).Address, opAsset.Code).Return(nil, sql.ErrNoRows)
			statsManager.On("UpdateGet", &paymentData, direction, now).Return(stats, nil).Once()
			v := NewOutgoingLimitsValidator(&paymentData, &statsManager, &histMock, limits, now)
			result, err := v.VerifyLimits()
			So(err, ShouldBeNil)
			assert.Equal(t, &results.ExceededLimitError{Description: fmt.Sprintf("Daily outgoing payments limit for anonymous account exceeded: %s out of %s %s.",
				amount.String(xdr.Int64(opAmount+opAmount)),
				amount.String(xdr.Int64(limits.MaxDailyOutcome)),
				opAsset.Code,
			)}, result)
		})
		Convey("Asset is anonymous exceeds monthly limit, with no account limits", func() {
			limits := config.AnonymousUserRestrictions{
				MaxDailyOutcome:   2 * opAmount,
				MaxMonthlyOutcome: 2*opAmount - 1,
			}
			paymentData.Asset.IsAnonymous = true
			histMock := history.QMock{}
			histMock.On("GetAccountLimits", paymentData.GetAccount(direction).Address, opAsset.Code).Return(nil, sql.ErrNoRows)
			statsManager.On("UpdateGet", &paymentData, direction, now).Return(stats, nil).Once()
			v := NewOutgoingLimitsValidator(&paymentData, &statsManager, &histMock, limits, now)
			result, err := v.VerifyLimits()
			So(err, ShouldBeNil)
			assert.Equal(t, &results.ExceededLimitError{Description: fmt.Sprintf("Monthly outgoing payments limit for anonymous account exceeded: %s out of %s %s.",
				amount.String(xdr.Int64(opAmount+opAmount)),
				amount.String(xdr.Int64(limits.MaxMonthlyOutcome)),
				opAsset.Code,
			)}, result)
		})
		Convey("Asset is anonymous exceeds annual limit, with no account limits", func() {
			limits := config.AnonymousUserRestrictions{
				MaxDailyOutcome:   2 * opAmount,
				MaxMonthlyOutcome: 2 * opAmount,
				MaxAnnualOutcome:  2*opAmount - 1,
			}
			paymentData.Asset.IsAnonymous = true
			histMock := history.QMock{}
			histMock.On("GetAccountLimits", paymentData.GetAccount(direction).Address, opAsset.Code).Return(nil, sql.ErrNoRows)
			statsManager.On("UpdateGet", &paymentData, direction, now).Return(stats, nil).Once()
			v := NewOutgoingLimitsValidator(&paymentData, &statsManager, &histMock, limits, now)
			result, err := v.VerifyLimits()
			So(err, ShouldBeNil)
			assert.Equal(t, &results.ExceededLimitError{Description: fmt.Sprintf("Annual outgoing payments limit for anonymous account exceeded: %s out of %s %s.",
				amount.String(xdr.Int64(opAmount+opAmount)),
				amount.String(xdr.Int64(limits.MaxAnnualOutcome)),
				opAsset.Code,
			)}, result)
		})
		Convey("Asset is anonymous to SettlementAgent, with no account limits", func() {
			limits := config.AnonymousUserRestrictions{
				MaxDailyOutcome:   2 * opAmount,
				MaxMonthlyOutcome: 2 * opAmount,
				MaxAnnualOutcome:  2*opAmount - 1,
			}
			paymentData.Asset.IsAnonymous = true
			histMock := history.QMock{}
			histMock.On("GetAccountLimits", paymentData.GetAccount(direction).Address, opAsset.Code).Return(nil, sql.ErrNoRows)
			statsManager.On("UpdateGet", &paymentData, direction, now).Return(stats, nil).Once()
			paymentData.GetCounterparty(direction).AccountType = xdr.AccountTypeAccountSettlementAgent
			v := NewOutgoingLimitsValidator(&paymentData, &statsManager, &histMock, limits, now)
			result, err := v.VerifyLimits()
			So(err, ShouldBeNil)
			So(result, ShouldBeNil)
		})
		Convey("Asset is anonymous to merchant", func() {
			limits := config.AnonymousUserRestrictions{
				MaxDailyOutcome:   2*opAmount - 1,
				MaxMonthlyOutcome: 2*opAmount - 1,
				MaxAnnualOutcome:  2 * opAmount,
			}
			paymentData.Asset.IsAnonymous = true
			histMock := history.QMock{}
			histMock.On("GetAccountLimits", paymentData.GetAccount(direction).Address, opAsset.Code).Return(nil, sql.ErrNoRows)
			statsManager.On("UpdateGet", &paymentData, direction, now).Return(stats, nil).Once()
			paymentData.GetCounterparty(direction).AccountType = xdr.AccountTypeAccountMerchant
			v := NewOutgoingLimitsValidator(&paymentData, &statsManager, &histMock, limits, now)
			result, err := v.VerifyLimits()
			So(err, ShouldBeNil)
			So(result, ShouldBeNil)
		})

	})
}
