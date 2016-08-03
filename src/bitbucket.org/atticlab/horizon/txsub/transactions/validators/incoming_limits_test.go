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
	"bitbucket.org/atticlab/horizon/txsub/transactions/statistics"
	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"time"
	"database/sql"
)

func TestIncomingLimits(t *testing.T) {
	log.DefaultLogger.Entry.Logger.Level = log.DebugLevel

	sourceKey, err := keypair.Random()
	assert.Nil(t, err)
	destKey, err := keypair.Random()
	assert.Nil(t, err)
	source := &core.Account{
		Accountid:   sourceKey.Address(),
		AccountType: xdr.AccountTypeAccountAnonymousUser,
	}
	destination := &core.Account{
		Accountid:   destKey.Address(),
		AccountType: xdr.AccountTypeAccountAnonymousUser,
	}
	opAmount := int64(amount.One * 100)
	opAsset := history.Asset{
		Code:        "UAH",
		IsAnonymous: false,
	}

	opData := statistics.NewOperationData(source, 0, "random_tx_hash")
	paymentData := statistics.NewPaymentData(destination, opAsset, opAmount, opData)
	direction := statistics.PaymentDirectionIncoming

	accountLimits := history.AccountLimits{
		Account:         paymentData.GetAccount(direction).Accountid,
		AssetCode:       opAsset.Code,
		MaxOperationOut: -1,
		DailyMaxOut:     -1,
		MonthlyMaxOut:   -1,
		MaxOperationIn:  -1,
		DailyMaxIn:      -1,
		MonthlyMaxIn:    -1,
	}

	accountTrustLine := core.Trustline{
		Accountid: paymentData.GetAccount(direction).Accountid,
		Balance:   0,
	}

	statsManager := &statistics.ManagerMock{}

	now := time.Now()
	Convey("Incoming limits test:", t, func() {
		Convey("No limits for account & asset is not anonymous", func() {
			histMock := history.QMock{}
			histMock.On("GetAccountLimits", paymentData.GetAccount(direction).Accountid, opAsset.Code).Return(nil, sql.ErrNoRows)
			v := NewIncomingLimitsValidator(&paymentData, accountTrustLine, &histMock, statsManager, config.AnonymousUserRestrictions{}, now)
			result, err := v.VerifyLimits()
			So(err, ShouldBeNil)
			So(result, ShouldBeNil)
		})
		Convey("All limits are empty for account & asset is not anonymous", func() {
			histMock := history.QMock{}
			limits := accountLimits
			histMock.On("GetAccountLimits", paymentData.GetAccount(direction).Accountid, opAsset.Code).Return(limits, nil)
			v := NewIncomingLimitsValidator(&paymentData, accountTrustLine, &histMock, statsManager, config.AnonymousUserRestrictions{}, now)
			result, err := v.VerifyLimits()
			So(err, ShouldBeNil)
			So(result, ShouldBeNil)
		})
		Convey("Asset is not anonymous, exceeds op amount", func() {
			limits := accountLimits
			limits.MaxOperationIn = opAmount - 1
			histMock := history.QMock{}
			histMock.On("GetAccountLimits", paymentData.GetAccount(direction).Accountid, opAsset.Code).Return(limits, nil)
			v := NewIncomingLimitsValidator(&paymentData, accountTrustLine, &histMock, statsManager, config.AnonymousUserRestrictions{}, now)
			result, err := v.VerifyLimits()
			So(err, ShouldBeNil)
			assert.Equal(t, &results.ExceededLimitError{Description: fmt.Sprintf(
				"Maximal operation amount for account (%s) exceeded: %s of %s %s",
				paymentData.GetAccount(direction).Accountid,
				amount.String(xdr.Int64(opAmount)),
				amount.String(xdr.Int64(limits.MaxOperationIn)),
				opAsset.Code,
			)}, result)
		})
		Convey("Asset is not anonymous, exceeds daily limit with stats", func() {
			limits := accountLimits
			limits.DailyMaxIn = 2*opAmount - 1
			histMock := history.QMock{}
			histMock.On("GetAccountLimits", paymentData.GetAccount(direction).Accountid, opAsset.Code).Return(limits, nil)
			stats := map[xdr.AccountType]history.AccountStatistics{
				paymentData.GetAccount(direction).AccountType: history.AccountStatistics{
					Account:          paymentData.GetAccount(direction).Accountid,
					AssetCode:        opAsset.Code,
					CounterpartyType: int16(paymentData.GetCounterparty(direction).AccountType),
					DailyIncome:      opAmount + opAmount,
				},
			}
			statsManager.On("UpdateGet", &paymentData, direction, now).Return(stats, nil).Once()
			v := NewIncomingLimitsValidator(&paymentData, accountTrustLine, &histMock, statsManager, config.AnonymousUserRestrictions{}, now)
			result, err := v.VerifyLimits()
			So(err, ShouldBeNil)
			assert.Equal(t, &results.ExceededLimitError{Description: fmt.Sprintf("Daily incoming payments limit for account exceeded: %s out of %s %s.",
				amount.String(xdr.Int64(opAmount + opAmount)),
				amount.String(xdr.Int64(limits.DailyMaxIn)),
				opAsset.Code,
			)}, result)
		})
		Convey("Asset is not anonymous, exceeds monthly limit with empty stats", func() {
			limits := accountLimits
			limits.MonthlyMaxIn = 2*opAmount - 1
			histMock := history.QMock{}
			histMock.On("GetAccountLimits", paymentData.GetAccount(direction).Accountid, opAsset.Code).Return(limits, nil)
			stats := map[xdr.AccountType]history.AccountStatistics{
				xdr.AccountTypeAccountAnonymousUser: history.AccountStatistics{
					Account:          paymentData.GetAccount(direction).Accountid,
					AssetCode:        opAsset.Code,
					CounterpartyType: int16(xdr.AccountTypeAccountSettlementAgent),
					MonthlyIncome:    opAmount + opAmount,
				},
			}
			statsManager.On("UpdateGet", &paymentData, direction, now).Return(stats, nil)
			v := NewIncomingLimitsValidator(&paymentData, accountTrustLine, &histMock, statsManager, config.AnonymousUserRestrictions{}, now)
			result, err := v.VerifyLimits()
			So(err, ShouldBeNil)
			assert.Equal(t, &results.ExceededLimitError{Description: fmt.Sprintf("Monthly incoming payments limit for account exceeded: %s out of %s %s.",
				amount.String(xdr.Int64(opAmount + opAmount)),
				amount.String(xdr.Int64(limits.MonthlyMaxIn)),
				opAsset.Code,
			)}, result)
		})
		stats := map[xdr.AccountType]history.AccountStatistics{
			xdr.AccountTypeAccountAnonymousUser: history.AccountStatistics{
				Account:          paymentData.GetAccount(direction).Accountid,
				AssetCode:        opAsset.Code,
				CounterpartyType: int16(xdr.AccountTypeAccountSettlementAgent),
				DailyIncome:      opAmount,
				MonthlyIncome:    opAmount,
				AnnualIncome:     opAmount,
			},
		}
		Convey("Asset is anonymous exceeds max balance", func() {
			limits := config.AnonymousUserRestrictions{
				MaxBalance: 2*opAmount - 1,
			}
			accountTrustLine.Balance = xdr.Int64(opAmount)
			paymentData.Asset.IsAnonymous = true
			histMock := history.QMock{}
			histMock.On("GetAccountLimits", paymentData.GetAccount(direction).Accountid, opAsset.Code).Return(nil, sql.ErrNoRows)
			statsManager.On("UpdateGet", &paymentData, direction, now).Return(stats, nil)
			v := NewIncomingLimitsValidator(&paymentData, accountTrustLine, &histMock, statsManager, limits, now)
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
			histMock.On("GetAccountLimits", paymentData.GetAccount(direction).Accountid, opAsset.Code).Return(nil, sql.ErrNoRows)
			statsManager.On("UpdateGet", &paymentData, direction, now).Return(stats, nil)
			paymentData.GetAccount(direction).AccountType = xdr.AccountTypeAccountMerchant
			v := NewIncomingLimitsValidator(&paymentData, accountTrustLine, &histMock, statsManager, limits, now)
			result, err := v.VerifyLimits()
			So(err, ShouldBeNil)
			So(result, ShouldBeNil)
		})

	})
}
