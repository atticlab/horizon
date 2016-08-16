package history

import (
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/db2"
	"bitbucket.org/atticlab/horizon/log"
	"github.com/stretchr/testify/mock"
	"math/rand"
	"time"
)

type QMock struct {
	mock.Mock
}

// GetAccountLimits returns limits row by account and asset.
func (m *QMock) GetAccountLimits(dest interface{}, address string, assetCode string) error {
	a := m.Called(address, assetCode)
	rawLimits := a.Get(0)
	if rawLimits != nil {
		limits := rawLimits.(AccountLimits)
		destLimits := dest.(*AccountLimits)
		*destLimits = limits
	}
	return a.Error(1)
}

// Inserts new account limits instance
func (m *QMock) CreateAccountLimits(limits AccountLimits) error {
	return m.Called(limits).Error(0)
}

// Updates account's limits
func (m *QMock) UpdateAccountLimits(limits AccountLimits) error {
	return m.Called(limits).Error(0)
}

// GetStatisticsByAccountAndAsset selects rows from `account_statistics` by address and asset code
func (m *QMock) GetStatisticsByAccountAndAsset(dest map[xdr.AccountType]AccountStatistics, addy string, assetCode string, now time.Time) error {
	a := m.Called(addy, assetCode, now)
	rawStats := a.Get(0)
	if rawStats != nil {
		for key, value := range rawStats.(map[xdr.AccountType]AccountStatistics) {
			dest[key] = value
		}
	}
	return a.Error(1)
}

// Traits

type AccountTraitsQMock struct {
	mock.Mock
}

func (q *AccountTraitsQMock) ForAccount(aid string) (traits AccountTraits, err error) {
	a := q.Called(aid)
	return a.Get(0).(AccountTraits), a.Error(1)
}
func (q *AccountTraitsQMock) ByID(id int64) (traits AccountTraits, err error) {
	a := q.Called(id)
	return a.Get(0).(AccountTraits), a.Error(1)
}
func (q *AccountTraitsQMock) Page(page db2.PageQuery) AccountTraitsQInterface {
	a := q.Called(page)
	return a.Get(0).(AccountTraitsQInterface)
}

func (q *AccountTraitsQMock) Select(dest interface{}) error {
	a := q.Called(dest)
	return a.Error(0)
}

func (m *QMock) AccountTraitsQ() AccountTraitsQInterface {
	a := m.Called()
	return a.Get(0).(AccountTraitsQInterface)
}

// Inserts new instance of account traits
func (m *QMock) InsertAccountTraits(traits AccountTraits) error {
	return m.Called(traits).Error(0)
}

// Updates account traits
func (m *QMock) UpdateAccountTraits(traits AccountTraits) error {
	return m.Called(traits).Error(0)
}

func (m *QMock) DeleteAccountTraits(id int64) error {
	return m.Called(id).Error(0)
}

func (m *QMock) Asset(dest interface{}, asset xdr.Asset) error {
	log.Debug("Asset is called")
	a := m.Called(asset)
	rawAsset := a.Get(0)
	if rawAsset != nil {
		log.Debug("Raw asset is not null")
		asset := a.Get(0).(Asset)
		destAsset := dest.(*Asset)
		*destAsset = asset
	}
	return a.Error(1)
}

// Deletes asset from db by id
func (m *QMock) DeleteAsset(id int64) (bool, error) {
	log.Panic("Not implemented")
	return false, nil
}

// updates asset
func (m *QMock) UpdateAsset(asset *Asset) (bool, error) {
	log.Panic("Not implemented")
	return false, nil
}

// inserts asset
func (m *QMock) InsertAsset(asset *Asset) (err error) {
	log.Panic("Not implemented")
	return nil
}

func (m *QMock) AccountByAddress(dest interface{}, addy string) error {
	a := m.Called(addy)
	rawAccount := a.Get(0)
	if rawAccount != nil {
		account := a.Get(0).(Account)
		destAccount := dest.(*Account)
		*destAccount = account
	}
	return a.Error(1)
}

// selects commission by id
func (m *QMock) CommissionByHash(hash string) (*Commission, error) {
	log.Panic("Not implemented")
	return nil, nil
}

// Inserts new commission
func (m *QMock) InsertCommission(commission *Commission) (err error) {
	log.Panic("Not implemented")
	return nil
}

// Deletes commission
func (m *QMock) DeleteCommission(hash string) (bool, error) {
	log.Panic("Not implemented")
	return false, nil
}

// update commission
func (m *QMock) UpdateCommission(commission *Commission) (bool, error) {
	log.Panic("Not implemented")
	return false, nil
}

func CreateRandomAccountStats(account string, counterpartyType xdr.AccountType, asset string) AccountStatistics {
	return CreateRandomAccountStatsWithMinValue(account, counterpartyType, asset, 0)
}

func CreateRandomAccountStatsWithMinValue(account string, counterpartyType xdr.AccountType, asset string, minValue int64) AccountStatistics {
	return AccountStatistics{
		Account:          account,
		AssetCode:        asset,
		CounterpartyType: int16(counterpartyType),
		DailyIncome:      Max(rand.Int63(), minValue),
		DailyOutcome:     Max(rand.Int63(), minValue),
		WeeklyIncome:     Max(rand.Int63(), minValue),
		WeeklyOutcome:    Max(rand.Int63(), minValue),
		MonthlyIncome:    Max(rand.Int63(), minValue),
		MonthlyOutcome:   Max(rand.Int63(), minValue),
		AnnualIncome:     Max(rand.Int63(), minValue),
		AnnualOutcome:    Max(rand.Int63(), minValue),
		UpdatedAt:        time.Unix(time.Now().Unix(), 0),
	}
}

func Max(x int64, y int64) int64 {
	if x > y {
		return x
	}
	return y
}
