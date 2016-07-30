package statistics

import (
	"bitbucket.org/atticlab/go-smart-base/keypair"
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/log"
	"bitbucket.org/atticlab/horizon/redis"
	"errors"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
	"time"
)

func TestStatistics(t *testing.T) {

	log.DefaultLogger.Entry.Logger.Level = log.DebugLevel

	counterparties := []xdr.AccountType{
		xdr.AccountTypeAccountAnonymousUser,
		xdr.AccountTypeAccountRegisteredUser,
		xdr.AccountTypeAccountMerchant,
		xdr.AccountTypeAccountDistributionAgent,
		xdr.AccountTypeAccountSettlementAgent,
		xdr.AccountTypeAccountExchangeAgent,
		xdr.AccountTypeAccountBank,
	}

	statsTimeOut := time.Duration(1) * time.Hour
	opTimeout := time.Duration(30) * time.Minute
	kp, err := keypair.Random()
	assert.Nil(t, err)
	account := kp.Address()
	assetCode := "USD"
	opAmount := int64(10000)
	now := time.Now()
	txHash := "random_tx_hash"
	counterparty := xdr.AccountTypeAccountMerchant
	opIndex := 1
	isIncome := true
	updatedTime := time.Now().AddDate(0, 0, -1)

	Convey("UpdateGet", t, func() {
		returnedStats := createRandomStats(account, assetCode, updatedTime, counterparties)

		historyQ := &history.QMock{}
		manager := NewManager(historyQ, counterparties, statsTimeOut, opTimeout)
		connProvider := &redis.ConnectionProviderMock{}
		conn := &redis.ConnectionMock{}
		conn.On("Close").Return(nil)
		connProvider.On("GetConnection").Return(conn)
		manager.connectionProvider = connProvider
		processedOpProvider := &redis.ProcessedOpProviderMock{}
		manager.processedOpProvider = processedOpProvider
		accountStatsProvider := &redis.AccountStatisticsProviderMock{}
		manager.accountStatsProvider = accountStatsProvider
		opKey := redis.GetProcessedOpKey(txHash, opIndex)

		Convey("Failed to watch", func() {
			errorData := "failed to watch op"
			conn.On("Watch", opKey).Return(errors.New(errorData)).Once()
			result, err := manager.UpdateGet(account, assetCode, counterparty, isIncome, now, txHash, opIndex, opAmount)
			So(err.Error(), ShouldEqual, errorData)
			So(result, ShouldBeNil)
		})
		conn.On("Watch", opKey).Return(nil)
		Convey("Op processed", func() {
			Convey("Failed to check if op was processed", func() {
				errorData := "Failed to check if op was processed"
				processedOpProvider.On("Get", txHash, opIndex).Return(nil, errors.New(errorData)).Once()
				result, err := manager.UpdateGet(account, assetCode, counterparty, isIncome, now, txHash, opIndex, opAmount)
				So(err.Error(), ShouldEqual, errorData)
				So(result, ShouldBeNil)
			})
			Convey("Op was processed", func() {
				processedOp := redis.NewProcessedOp(txHash, opIndex, opAmount, now)
				processedOpProvider.On("Get", txHash, opIndex).Return(processedOp, nil)
				Convey("Failed to unwatch", func() {
					errorData := "failed to connect"
					conn.On("UnWatch").Return(errors.New(errorData)).Once()
					result, err := manager.UpdateGet(account, assetCode, counterparty, isIncome, now, txHash, opIndex, opAmount)
					So(err.Error(), ShouldEqual, errorData)
					So(result, ShouldBeNil)
				})
				conn.On("UnWatch").Return(nil)
				Convey("Got stats from redis", func() {
					accountStatsProvider.On("Get", account, assetCode, counterparties).Return(&returnedStats, nil).Once()
					result, err := manager.UpdateGet(account, assetCode, counterparty, isIncome, now, txHash, opIndex, opAmount)
					So(err, ShouldBeNil)
					assert.Equal(t, returnedStats.AccountsStatistics, result)
				})
				Convey("Got stats from history", func() {
					accountStatsProvider.On("Get", account, assetCode, counterparties).Return(nil, nil).Once()
					historyQ.On("GetStatisticsByAccountAndAsset", account, assetCode, now).Return(returnedStats.AccountsStatistics, nil)
					accountStatsProvider.On("Insert", &returnedStats, statsTimeOut).Return(nil)
					result, err := manager.UpdateGet(account, assetCode, counterparty, isIncome, now, txHash, opIndex, opAmount)
					So(err, ShouldBeNil)
					assert.Equal(t, returnedStats.AccountsStatistics, result)
				})
			})
		})
		Convey("Op not processed", func() {
			processedOpProvider.On("Get", txHash, opIndex).Return(nil, nil)
			statsKey := redis.GetAccountStatisticsKey(account, assetCode)
			Convey("Failed to watch stats", func() {
				errorData := "failed to watch stats"
				conn.On("Watch", statsKey).Return(errors.New(errorData))
				result, err := manager.UpdateGet(account, assetCode, counterparty, isIncome, now, txHash, opIndex, opAmount)
				So(err.Error(), ShouldEqual, errorData)
				So(result, ShouldBeNil)
			})
			conn.On("Watch", statsKey).Return(nil)
			Convey("Failed to get stats from redis", func() {
				errorData := "Failed to get stats from redis"
				accountStatsProvider.On("Get", account, assetCode, counterparties).Return(nil, errors.New(errorData))
				result, err := manager.UpdateGet(account, assetCode, counterparty, isIncome, now, txHash, opIndex, opAmount)
				So(err.Error(), ShouldEqual, errorData)
				So(result, ShouldBeNil)
			})
			Convey("Redis stats are empty - get from db", func() {
				accountStatsProvider.On("Get", account, assetCode, counterparties).Return(nil, nil)
				Convey("Failed to get stats from db", func() {
					errorData := "Failed to get stats from history"
					historyQ.On("GetStatisticsByAccountAndAsset", account, assetCode, now).Return(nil, errors.New(errorData))
					result, err := manager.UpdateGet(account, assetCode, counterparty, isIncome, now, txHash, opIndex, opAmount)
					So(err.Error(), ShouldEqual, errorData)
					So(result, ShouldBeNil)
				})
			})
			Convey("Account stats cleared", func() {
				expectedStats := copyAccountStats(&returnedStats)
				_, ok := expectedStats.AccountsStatistics[counterparty]
				if !ok {
					expectedStats.AccountsStatistics[counterparty] = history.NewAccountStatistics(expectedStats.Account, expectedStats.AssetCode, counterparty)
				}
				for key, value := range expectedStats.AccountsStatistics {
					value.ClearObsoleteStats(now)
					if key == counterparty {
						value.Update(opAmount, now, now, isIncome)
					}
					expectedStats.AccountsStatistics[key] = value
				}
				accountStatsProvider.On("Get", account, assetCode, counterparties).Return(&returnedStats, nil).Once()
				Convey("Multi failed", func() {
					errorData := "Failed to start multi"
					conn.On("Multi").Return(errors.New(errorData))
					result, err := manager.UpdateGet(account, assetCode, counterparty, isIncome, now, txHash, opIndex, opAmount)
					So(err.Error(), ShouldEqual, errorData)
					So(result, ShouldBeNil)
				})
				conn.On("Multi").Return(nil)
				Convey("Failed to insert stats", func() {
					errorData := "Failed to insert stats"
					accountStatsProvider.On("Insert", expectedStats, statsTimeOut).Return(errors.New(errorData)).Once()
					result, err := manager.UpdateGet(account, assetCode, counterparty, isIncome, now, txHash, opIndex, opAmount)
					So(err.Error(), ShouldEqual, errorData)
					So(result, ShouldBeNil)
				})
				accountStatsProvider.On("Insert", expectedStats, statsTimeOut).Return(nil).Once()
				processedOp := redis.NewProcessedOp(txHash, opIndex, opAmount, now)
				Convey("Failed to insert op processed", func() {
					errorData := "failed to insert op processed"
					processedOpProvider.On("Insert", processedOp, opTimeout).Return(errors.New(errorData))
					result, err := manager.UpdateGet(account, assetCode, counterparty, isIncome, now, txHash, opIndex, opAmount)
					So(err.Error(), ShouldEqual, errorData)
					So(result, ShouldBeNil)
				})
				processedOpProvider.On("Insert", processedOp, opTimeout).Return(nil)
				Convey("Failed to exec", func() {
					errorData := "failed to exec"
					conn.On("Exec").Return(false, errors.New(errorData))
					result, err := manager.UpdateGet(account, assetCode, counterparty, isIncome, now, txHash, opIndex, opAmount)
					So(err.Error(), ShouldEqual, errorData)
					So(result, ShouldBeNil)
				})
				Convey("Retries", func() {
					conn.On("Exec").Return(false, nil)
					result, retry, err := manager.updateGet(account, assetCode, counterparty, isIncome, now, txHash, opIndex, opAmount)
					So(err, ShouldBeNil)
					So(retry, ShouldBeTrue)
					So(result, ShouldBeNil)
				})
			})
		})
	})

	Convey("CancelOp", t, func() {
		returnedStats := createRandomStatsWithMinValue(account, assetCode, updatedTime, counterparties, opAmount)

		historyQ := &history.QMock{}
		manager := NewManager(historyQ, counterparties, statsTimeOut, opTimeout)
		connProvider := &redis.ConnectionProviderMock{}
		conn := &redis.ConnectionMock{}
		conn.On("Close").Return(nil)
		connProvider.On("GetConnection").Return(conn)
		manager.connectionProvider = connProvider
		processedOpProvider := &redis.ProcessedOpProviderMock{}
		manager.processedOpProvider = processedOpProvider
		accountStatsProvider := &redis.AccountStatisticsProviderMock{}
		manager.accountStatsProvider = accountStatsProvider
		opKey := redis.GetProcessedOpKey(txHash, opIndex)

		Convey("Failed to watch", func() {
			errorData := "failed to watch op"
			conn.On("Watch", opKey).Return(errors.New(errorData)).Once()
			err = manager.CancelOp(account, assetCode, counterparty, isIncome, now, txHash, opIndex, opAmount)
			So(err.Error(), ShouldEqual, errorData)
		})
		conn.On("Watch", opKey).Return(nil)
		Convey("Failed to check if op was processed", func() {
			errorData := "Failed to check if op was processed"
			processedOpProvider.On("Get", txHash, opIndex).Return(nil, errors.New(errorData)).Once()
			err := manager.CancelOp(account, assetCode, counterparty, isIncome, now, txHash, opIndex, opAmount)
			So(err.Error(), ShouldEqual, errorData)
		})
		Convey("Op was already canceled", func() {
			processedOpProvider.On("Get", txHash, opIndex).Return(nil, nil)
			Convey("Failed to unwatch", func() {
				errorData := "failed to connect"
				conn.On("UnWatch").Return(errors.New(errorData)).Once()
				err := manager.CancelOp(account, assetCode, counterparty, isIncome, now, txHash, opIndex, opAmount)
				So(err.Error(), ShouldEqual, errorData)
			})
			conn.On("UnWatch").Return(nil)
			err := manager.CancelOp(account, assetCode, counterparty, isIncome, now, txHash, opIndex, opAmount)
			So(err, ShouldBeNil)
		})
		processedOp := redis.NewProcessedOp(txHash, opIndex, opAmount, now.AddDate(0, 0, -1))
		processedOpProvider.On("Get", txHash, opIndex).Return(processedOp, nil)
		Convey("Failed to watch stats", func() {
			errorData := "failed to watch stats"
			conn.On("Watch", returnedStats.GetKey()).Return(errors.New(errorData)).Once()
			err = manager.CancelOp(account, assetCode, counterparty, isIncome, now, txHash, opIndex, opAmount)
			So(err.Error(), ShouldEqual, errorData)
		})
		conn.On("Watch", returnedStats.GetKey()).Return(nil)
		Convey("No stats in redis", func() {
			accountStatsProvider.On("Get", account, assetCode, counterparties).Return(nil, nil).Once()
			conn.On("UnWatch").Return(nil)
			err := manager.CancelOp(account, assetCode, counterparty, isIncome, now, txHash, opIndex, opAmount)
			So(err, ShouldBeNil)
		})
		accountStatsProvider.On("Get", account, assetCode, counterparties).Return(&returnedStats, nil).Once()
		Convey("Multi failed", func() {
			errorData := "Failed to start multi"
			conn.On("Multi").Return(errors.New(errorData))
			err := manager.CancelOp(account, assetCode, counterparty, isIncome, now, txHash, opIndex, opAmount)
			So(err.Error(), ShouldEqual, errorData)
		})
		conn.On("Multi").Return(nil)
		expectedStats := copyAccountStats(&returnedStats)
		for key, value := range expectedStats.AccountsStatistics {
			value.ClearObsoleteStats(now)
			if key == counterparty {
				So(value.DailyIncome, ShouldEqual, 0)
				So(value.DailyOutcome, ShouldEqual, 0)
				value.Update(-opAmount, processedOp.TimeUpdated, now, isIncome)
				// op was added day ago, so Daily stats were cleared, but must be negative even with canceling
				So(value.DailyIncome, ShouldEqual, 0)
				So(value.DailyOutcome, ShouldEqual, 0)
			}
			expectedStats.AccountsStatistics[key] = value
		}
		Convey("Failed to insert stats", func() {
			errorData := "Failed to insert stats"
			accountStatsProvider.On("Insert", expectedStats, statsTimeOut).Return(errors.New(errorData)).Once()
			err := manager.CancelOp(account, assetCode, counterparty, isIncome, now, txHash, opIndex, opAmount)
			So(err.Error(), ShouldEqual, errorData)
		})
		accountStatsProvider.On("Insert", expectedStats, statsTimeOut).Return(nil).Once()
		Convey("Failed to delete op processed", func() {
			errorData := "failed to delete op processed"
			processedOpProvider.On("Delete", txHash, opIndex).Return(errors.New(errorData))
			err := manager.CancelOp(account, assetCode, counterparty, isIncome, now, txHash, opIndex, opAmount)
			So(err.Error(), ShouldEqual, errorData)
		})
		processedOpProvider.On("Delete", txHash, opIndex).Return(nil)
		Convey("Failed to exec", func() {
			errorData := "failed to exec"
			conn.On("Exec").Return(false, errors.New(errorData))
			err := manager.CancelOp(account, assetCode, counterparty, isIncome, now, txHash, opIndex, opAmount)
			So(err.Error(), ShouldEqual, errorData)
		})
	})
}

func copyAccountStats(source *redis.AccountStatistics) *redis.AccountStatistics {
	result := new(redis.AccountStatistics)
	*result = *source
	result.AccountsStatistics = make(map[xdr.AccountType]history.AccountStatistics)
	for key, value := range source.AccountsStatistics {
		result.AccountsStatistics[key] = value
	}
	return result
}

func createRandomStats(account, assetCode string, timeUpdated time.Time, counterparties []xdr.AccountType) redis.AccountStatistics {
	return createRandomStatsWithMinValue(account, assetCode, timeUpdated, counterparties, 0)
}

func createRandomStatsWithMinValue(account, assetCode string, timeUpdated time.Time, counterparties []xdr.AccountType, minValue int64) redis.AccountStatistics {
	stats := redis.NewAccountStatistics(account, assetCode, make(map[xdr.AccountType]history.AccountStatistics))
	for _, counterparty := range counterparties {
		if rand.Float32() < 0.5 {
			continue
		}
		stat := history.CreateRandomAccountStatsWithMinValue(account, counterparty, assetCode, minValue)
		stat.UpdatedAt = timeUpdated
		stats.AccountsStatistics[counterparty] = stat
	}
	return *stats
}
