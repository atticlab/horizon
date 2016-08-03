package statistics

import (
	"bitbucket.org/atticlab/go-smart-base/amount"
	"bitbucket.org/atticlab/go-smart-base/keypair"
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/accounttypes"
	"bitbucket.org/atticlab/horizon/db2/core"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/log"
	"bitbucket.org/atticlab/horizon/redis"
	"errors"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"math/rand"
	"testing"
	"time"
)

func TestStatistics(t *testing.T) {

	log.DefaultLogger.Entry.Logger.Level = log.DebugLevel

	counterparties := accounttype.GetAll()

	statsTimeOut := time.Duration(1) * time.Hour
	opTimeout := time.Duration(30) * time.Minute
	sourceKP, err := keypair.Random()
	assert.Nil(t, err)
	destKP, err := keypair.Random()
	assert.Nil(t, err)
	now := time.Now()
	operationData := NewOperationData(&core.Account{
		Accountid:   sourceKP.Address(),
		AccountType: xdr.AccountTypeAccountBank,
	}, 1, "random_tx_hash")
	paymentData := NewPaymentData(&core.Account{
		Accountid:   destKP.Address(),
		AccountType: xdr.AccountTypeAccountAnonymousUser,
	}, history.Asset{
		Code: "UAH",
	}, 100*amount.One, operationData)
	opIndex := 1
	direction := PaymentDirectionIncoming
	isIncome := direction.IsIncoming()
	updatedTime := time.Now().AddDate(0, 0, -1)
	account := paymentData.GetAccount(direction).Accountid
	assetCode := paymentData.Asset.Code

	Convey("UpdateGet", t, func() {
		historyQ := &history.QMock{}
		manager := NewManager(historyQ, counterparties)
		manager.SetProcessedOpTimeout(opTimeout)
		manager.SetStatisticsTimeout(statsTimeOut)
		connProvider := &redis.ConnectionProviderMock{}
		conn := &redis.ConnectionMock{}
		conn.On("Close").Return(nil)
		connProvider.On("GetConnection").Return(conn)
		manager.connectionProvider = connProvider
		processedOpProvider := &redis.ProcessedOpProviderMock{}
		manager.processedOpProvider = processedOpProvider
		accountStatsProvider := &redis.AccountStatisticsProviderMock{}
		manager.accountStatsProvider = accountStatsProvider
		opKey := redis.GetProcessedOpKey(paymentData.TxHash, paymentData.Index, isIncome)

		Convey("Failed to watch", func() {
			errorData := "failed to watch op"
			conn.On("Watch", opKey).Return(errors.New(errorData)).Once()
			result, err := manager.UpdateGet(&paymentData, direction, now)
			So(err.Error(), ShouldEqual, errorData)
			So(result, ShouldBeNil)
		})
		conn.On("Watch", opKey).Return(nil)
		Convey("Op processed", func() {
			Convey("Failed to check if op was processed", func() {
				errorData := "Failed to check if op was processed"
				processedOpProvider.On("Get", paymentData.TxHash, opIndex, isIncome).Return(nil, errors.New(errorData)).Once()
				result, err := manager.UpdateGet(&paymentData, direction, now)
				So(err.Error(), ShouldEqual, errorData)
				So(result, ShouldBeNil)
			})
			Convey("Op was processed", func() {
				processedOp := redis.NewProcessedOp(paymentData.TxHash, opIndex, paymentData.Amount, isIncome, now)
				processedOpProvider.On("Get", paymentData.TxHash, opIndex, isIncome).Return(processedOp, nil)
				Convey("Failed to unwatch", func() {
					errorData := "failed to connect"
					conn.On("UnWatch").Return(errors.New(errorData)).Once()
					result, err := manager.UpdateGet(&paymentData, direction, now)
					So(err.Error(), ShouldEqual, errorData)
					So(result, ShouldBeNil)
				})
				conn.On("UnWatch").Return(nil)
				returnedStats := createRandomStats(paymentData.GetAccount(direction).Accountid, paymentData.Asset.Code, updatedTime, counterparties)
				Convey("Got stats from redis", func() {
					accountStatsProvider.On("Get", account, assetCode, counterparties).Return(&returnedStats, nil).Once()
					result, err := manager.UpdateGet(&paymentData, direction, now)
					So(err, ShouldBeNil)
					assert.Equal(t, returnedStats.AccountsStatistics, result)
				})
				Convey("Got stats from history", func() {
					accountStatsProvider.On("Get", account, assetCode, counterparties).Return(nil, nil).Once()
					historyQ.On("GetStatisticsByAccountAndAsset", account, assetCode, now).Return(returnedStats.AccountsStatistics, nil)
					accountStatsProvider.On("Insert", &returnedStats, statsTimeOut).Return(nil)
					result, err := manager.UpdateGet(&paymentData, direction, now)
					So(err, ShouldBeNil)
					assert.Equal(t, returnedStats.AccountsStatistics, result)
				})
			})
		})
		Convey("Op not processed", func() {
			processedOpProvider.On("Get", paymentData.TxHash, opIndex, isIncome).Return(nil, nil)
			statsKey := redis.GetAccountStatisticsKey(account, assetCode)
			Convey("Failed to watch stats", func() {
				errorData := "failed to watch stats"
				conn.On("Watch", statsKey).Return(errors.New(errorData))
				result, err := manager.UpdateGet(&paymentData, direction, now)
				So(err.Error(), ShouldEqual, errorData)
				So(result, ShouldBeNil)
			})
			conn.On("Watch", statsKey).Return(nil)
			Convey("Failed to get stats from redis", func() {
				errorData := "Failed to get stats from redis"
				accountStatsProvider.On("Get", account, assetCode, counterparties).Return(nil, errors.New(errorData))
				result, err := manager.UpdateGet(&paymentData, direction, now)
				So(err.Error(), ShouldEqual, errorData)
				So(result, ShouldBeNil)
			})
			Convey("Redis stats are empty - get from db", func() {
				accountStatsProvider.On("Get", account, assetCode, counterparties).Return(nil, nil)
				Convey("Failed to get stats from db", func() {
					errorData := "Failed to get stats from history"
					historyQ.On("GetStatisticsByAccountAndAsset", account, assetCode, now).Return(nil, errors.New(errorData))
					result, err := manager.UpdateGet(&paymentData, direction, now)
					So(err.Error(), ShouldEqual, errorData)
					So(result, ShouldBeNil)
				})
			})
			Convey("Account stats cleared", func() {
				returnedStats := createRandomStats(paymentData.GetAccount(direction).Accountid, paymentData.Asset.Code, updatedTime, counterparties)
				expectedStats := copyAccountStats(&returnedStats)
				counterparty := paymentData.GetCounterparty(direction).AccountType
				if _, ok := expectedStats.AccountsStatistics[counterparty]; !ok {
					expectedStats.AccountsStatistics[counterparty] = history.NewAccountStatistics(expectedStats.Account, expectedStats.AssetCode, counterparty)
				}
				for key, value := range expectedStats.AccountsStatistics {
					value.ClearObsoleteStats(now)
					if key == counterparty {
						value.Update(paymentData.Amount, now, now, isIncome)
					}
					expectedStats.AccountsStatistics[key] = value
				}
				accountStatsProvider.On("Get", account, assetCode, counterparties).Return(&returnedStats, nil).Once()
				Convey("Multi failed", func() {
					errorData := "Failed to start multi"
					conn.On("Multi").Return(errors.New(errorData))
					result, err := manager.UpdateGet(&paymentData, direction, now)
					So(err.Error(), ShouldEqual, errorData)
					So(result, ShouldBeNil)
				})
				conn.On("Multi").Return(nil)
				Convey("Failed to insert stats", func() {
					errorData := "Failed to insert stats"
					accountStatsProvider.On("Insert", mock.Anything, statsTimeOut).Return(errors.New(errorData)).Run(func(args mock.Arguments) {
						actual := args.Get(0).(*redis.AccountStatistics)
						assert.Equal(t, len(actual.AccountsStatistics), len(expectedStats.AccountsStatistics))
						for aKey, aValue := range actual.AccountsStatistics {
							assert.Equal(t, expectedStats.AccountsStatistics[aKey], aValue)
						}
					}).Once()
					result, err := manager.UpdateGet(&paymentData, direction, now)
					So(err.Error(), ShouldEqual, errorData)
					So(result, ShouldBeNil)
				})
				accountStatsProvider.On("Insert", expectedStats, statsTimeOut).Return(nil).Once()
				processedOp := redis.NewProcessedOp(paymentData.TxHash, opIndex, paymentData.Amount, isIncome, now)
				Convey("Failed to insert op processed", func() {
					errorData := "failed to insert op processed"
					processedOpProvider.On("Insert", processedOp, opTimeout).Return(errors.New(errorData))
					result, err := manager.UpdateGet(&paymentData, direction, now)
					So(err.Error(), ShouldEqual, errorData)
					So(result, ShouldBeNil)
				})
				processedOpProvider.On("Insert", processedOp, opTimeout).Return(nil)
				Convey("Failed to exec", func() {
					errorData := "failed to exec"
					conn.On("Exec").Return(false, errors.New(errorData))
					result, err := manager.UpdateGet(&paymentData, direction, now)
					So(err.Error(), ShouldEqual, errorData)
					So(result, ShouldBeNil)
				})
				Convey("Retries", func() {
					conn.On("Exec").Return(false, nil)
					result, retry, err := manager.updateGet(&paymentData, direction, now)
					So(err, ShouldBeNil)
					So(retry, ShouldBeTrue)
					So(result, ShouldBeNil)
				})
			})
		})
	})

	Convey("CancelOp", t, func() {
		returnedStats := createRandomStatsWithMinValue(account, assetCode, updatedTime, counterparties, paymentData.Amount)

		historyQ := &history.QMock{}
		manager := NewManager(historyQ, counterparties)
		manager.SetProcessedOpTimeout(opTimeout)
		manager.SetStatisticsTimeout(statsTimeOut)
		connProvider := &redis.ConnectionProviderMock{}
		conn := &redis.ConnectionMock{}
		conn.On("Close").Return(nil)
		connProvider.On("GetConnection").Return(conn)
		manager.connectionProvider = connProvider
		processedOpProvider := &redis.ProcessedOpProviderMock{}
		manager.processedOpProvider = processedOpProvider
		accountStatsProvider := &redis.AccountStatisticsProviderMock{}
		manager.accountStatsProvider = accountStatsProvider
		opKey := redis.GetProcessedOpKey(paymentData.TxHash, opIndex, isIncome)

		Convey("Failed to watch", func() {
			errorData := "failed to watch op"
			conn.On("Watch", opKey).Return(errors.New(errorData)).Once()
			err = manager.CancelOp(&paymentData, direction, now)
			So(err.Error(), ShouldEqual, errorData)
		})
		conn.On("Watch", opKey).Return(nil)
		Convey("Failed to check if op was processed", func() {
			errorData := "Failed to check if op was processed"
			processedOpProvider.On("Get", paymentData.TxHash, opIndex, isIncome).Return(nil, errors.New(errorData)).Once()
			err := manager.CancelOp(&paymentData, direction, now)
			So(err.Error(), ShouldEqual, errorData)
		})
		Convey("Op was already canceled", func() {
			processedOpProvider.On("Get", paymentData.TxHash, opIndex, isIncome).Return(nil, nil)
			Convey("Failed to unwatch", func() {
				errorData := "failed to connect"
				conn.On("UnWatch").Return(errors.New(errorData)).Once()
				err := manager.CancelOp(&paymentData, direction, now)
				So(err.Error(), ShouldEqual, errorData)
			})
			conn.On("UnWatch").Return(nil)
			err := manager.CancelOp(&paymentData, direction, now)
			So(err, ShouldBeNil)
		})
		processedOp := redis.NewProcessedOp(paymentData.TxHash, opIndex, paymentData.Amount, isIncome, now.AddDate(0, 0, -1))
		processedOpProvider.On("Get", paymentData.TxHash, opIndex, isIncome).Return(processedOp, nil)
		Convey("Failed to watch stats", func() {
			errorData := "failed to watch stats"
			conn.On("Watch", returnedStats.GetKey()).Return(errors.New(errorData)).Once()
			err = manager.CancelOp(&paymentData, direction, now)
			So(err.Error(), ShouldEqual, errorData)
		})
		conn.On("Watch", returnedStats.GetKey()).Return(nil)
		Convey("No stats in redis", func() {
			accountStatsProvider.On("Get", account, assetCode, counterparties).Return(nil, nil).Once()
			conn.On("UnWatch").Return(nil)
			err := manager.CancelOp(&paymentData, direction, now)
			So(err, ShouldBeNil)
		})
		accountStatsProvider.On("Get", account, assetCode, counterparties).Return(&returnedStats, nil).Once()
		Convey("Multi failed", func() {
			errorData := "Failed to start multi"
			conn.On("Multi").Return(errors.New(errorData))
			err := manager.CancelOp(&paymentData, direction, now)
			So(err.Error(), ShouldEqual, errorData)
		})
		conn.On("Multi").Return(nil)
		expectedStats := copyAccountStats(&returnedStats)
		counterparty := paymentData.GetCounterparty(direction).AccountType
		for key, value := range expectedStats.AccountsStatistics {
			value.ClearObsoleteStats(now)
			if key == counterparty {
				So(value.DailyIncome, ShouldEqual, 0)
				So(value.DailyOutcome, ShouldEqual, 0)
				value.Update(-paymentData.Amount, processedOp.TimeUpdated, now, isIncome)
				// op was added day ago, so Daily stats were cleared, but must be negative even with canceling
				So(value.DailyIncome, ShouldEqual, 0)
				So(value.DailyOutcome, ShouldEqual, 0)
			}
			expectedStats.AccountsStatistics[key] = value
		}
		Convey("Failed to insert stats", func() {
			errorData := "Failed to insert stats"
			accountStatsProvider.On("Insert", expectedStats, statsTimeOut).Return(errors.New(errorData)).Once()
			err := manager.CancelOp(&paymentData, direction, now)
			So(err.Error(), ShouldEqual, errorData)
		})
		accountStatsProvider.On("Insert", expectedStats, statsTimeOut).Return(nil).Once()
		Convey("Failed to delete op processed", func() {
			errorData := "failed to delete op processed"
			processedOpProvider.On("Delete", paymentData.TxHash, opIndex, isIncome).Return(errors.New(errorData))
			err := manager.CancelOp(&paymentData, direction, now)
			So(err.Error(), ShouldEqual, errorData)
		})
		processedOpProvider.On("Delete", paymentData.TxHash, opIndex, isIncome).Return(nil)
		Convey("Failed to exec", func() {
			errorData := "failed to exec"
			conn.On("Exec").Return(false, errors.New(errorData))
			err := manager.CancelOp(&paymentData, direction, now)
			So(err.Error(), ShouldEqual, errorData)
		})
	})
	Convey("updateStats", t, func() {
		accountKP, err := keypair.Random()
		So(err, ShouldBeNil)
		account := accountKP.Address()
		assetCode := "AUAH"
		now := time.Date(1, 1, 2, 0, 0, 0, 0, time.Local)
		updatedTime := now.AddDate(0, 0, -1)
		opAmount := rand.Int63()
		isIncome := true
		counterparty := counterparties[rand.Intn(len(counterparties))]
		actual := createRandomStatsWithMinValue(account, assetCode, updatedTime, counterparties, opAmount)
		delete(actual.AccountsStatistics, counterparty)
		expected := copyAccountStats(&actual)
		if _, ok := expected.AccountsStatistics[counterparty]; !ok {
			expected.AccountsStatistics[counterparty] = history.NewAccountStatistics(expected.Account, expected.AssetCode, counterparty)
		}
		for key, value := range expected.AccountsStatistics {
			value.DailyIncome = 0
			value.DailyOutcome = 0
			value.UpdatedAt = now
			if key == counterparty {
				value.Update(opAmount, now, now, isIncome)
			}
			expected.AccountsStatistics[key] = value
		}
		m := NewManager(nil, counterparties)
		m.updateStats(&actual, counterparty, isIncome, opAmount, now)
		assert.Equal(t, expected, &actual)
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
