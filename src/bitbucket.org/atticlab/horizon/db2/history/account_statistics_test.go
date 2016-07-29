package history

import (
	"testing"

	"bitbucket.org/atticlab/go-smart-base/keypair"
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/test"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"time"
)

func TestAccountStatistics(t *testing.T) {
	tt := test.Start(t).Scenario("base")
	defer tt.Finish()

	account, err := keypair.Random()
	assert.Nil(t, err)
	assetCode := "USD"
	Convey("ClearObsoleteStats:", t, func() {
		updatedTime := time.Time{}.In(time.UTC)
		expectedStats := AccountStatistics{
			Account:          account.Address(),
			AssetCode:        assetCode,
			CounterpartyType: 1,
			DailyIncome:      10,
			DailyOutcome:     20,
			WeeklyIncome:     70,
			WeeklyOutcome:    140,
			MonthlyIncome:    280,
			MonthlyOutcome:   560,
			AnnualIncome:     3650,
			AnnualOutcome:    3650 * 2,
			UpdatedAt:        updatedTime,
		}
		actualStats := expectedStats
		actualStats.ClearObsoleteStats(updatedTime)
		actualStats.UpdatedAt = updatedTime
		assert.Equal(t, expectedStats, actualStats)
		expectedStats.DailyIncome = 0
		expectedStats.DailyOutcome = 0
		Convey("ClearObsoleteStats: daily", func() {
			actualStats.ClearObsoleteStats(updatedTime.AddDate(0, 0, 1))
			actualStats.UpdatedAt = updatedTime
			assert.Equal(t, expectedStats, actualStats)
		})
		expectedStats.WeeklyIncome = 0
		expectedStats.WeeklyOutcome = 0
		Convey("ClearObsoleteStats: weekly", func() {
			actualStats.ClearObsoleteStats(updatedTime.AddDate(0, 0, 7))
			actualStats.UpdatedAt = updatedTime
			assert.Equal(t, expectedStats, actualStats)
		})
		expectedStats.MonthlyIncome = 0
		expectedStats.MonthlyOutcome = 0
		Convey("ClearObsoleteStats: monthly", func() {
			actualStats.ClearObsoleteStats(updatedTime.AddDate(0, 1, 0))
			actualStats.UpdatedAt = updatedTime
			assert.Equal(t, expectedStats, actualStats)
		})
		expectedStats.AnnualIncome = 0
		expectedStats.AnnualOutcome = 0
		Convey("ClearObsoleteStats: yearly", func() {
			actualStats.ClearObsoleteStats(updatedTime.AddDate(1, 0, 0))
			actualStats.UpdatedAt = updatedTime
			assert.Equal(t, expectedStats, actualStats)
		})
	})
	Convey("GetStatisticsByAccountAndAsset", t, func() {
		q := &Q{tt.HorizonRepo()}
		asset := "USD"
		accountTypes := []xdr.AccountType{
			xdr.AccountTypeAccountAnonymousUser,
			xdr.AccountTypeAccountRegisteredUser,
			xdr.AccountTypeAccountMerchant,
			xdr.AccountTypeAccountDistributionAgent,
			xdr.AccountTypeAccountSettlementAgent,
			xdr.AccountTypeAccountExchangeAgent,
			xdr.AccountTypeAccountBank,
		}
		stats := make(map[xdr.AccountType]AccountStatistics)
		for _, t := range accountTypes {
			stats[t] = createRandomAccountStats(account.Address(), t, asset)
			err := q.CreateStats(stats[t])
			So(err, ShouldBeNil)
		}
		storedStats := make(map[xdr.AccountType]AccountStatistics)
		err = q.GetStatisticsByAccountAndAsset(storedStats, account.Address(), asset)
		So(err, ShouldBeNil)
		for key, value := range storedStats {
			stat, ok := stats[key]
			So(ok, ShouldBeTrue)
			stat.UpdatedAt = value.UpdatedAt
			assert.Equal(t, stat, value)
		}
	})
}

func createRandomAccountStats(account string, counterpartyType xdr.AccountType, asset string) AccountStatistics {
	return AccountStatistics{
		Account:          account,
		AssetCode:        asset,
		CounterpartyType: int16(counterpartyType),
		DailyIncome:      rand.Int63(),
		DailyOutcome:     rand.Int63(),
		WeeklyIncome:     rand.Int63(),
		WeeklyOutcome:    rand.Int63(),
		MonthlyIncome:    rand.Int63(),
		MonthlyOutcome:   rand.Int63(),
		AnnualIncome:     rand.Int63(),
		AnnualOutcome:    rand.Int63(),
		UpdatedAt: time.Now(),
	}
}
