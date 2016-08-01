package history

import (
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/log"
	"github.com/stretchr/testify/mock"
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
func (m *QMock) GetStatisticsByAccountAndAsset(dest map[xdr.AccountType]AccountStatistics, addy string, assetCode string) error {
	a := m.Called(addy, assetCode)
	rawStats := a.Get(0)
	if rawStats != nil {
		for key, value := range rawStats.(map[xdr.AccountType]AccountStatistics) {
			dest[key] = value
		}
	}
	return a.Error(1)
}

// Returns account traits instance by history.account id
func (m *QMock) GetAccountTraits(dest interface{}, id int64) error {
	log.Panic("Not implemented")
	return nil
}

// Inserts new instance of account traits
func (m *QMock) CreateAccountTraits(traits AccountTraits) error {
	return m.Called(traits).Error(0)
}

// Updates account traits
func (m *QMock) UpdateAccountTraits(traits AccountTraits) error {
	return m.Called(traits).Error(0)
}

// GetAccountTraitsByAddress returns traits for specified account
func (m *QMock) GetAccountTraitsByAddress(dest interface{}, accountID string) error {
	a := m.Called(accountID)
	rawTraits := a.Get(0)
	if rawTraits != nil {
		traits := a.Get(0).(AccountTraits)
		destTraits := dest.(*AccountTraits)
		*destTraits = traits
	}
	return a.Error(1)
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
