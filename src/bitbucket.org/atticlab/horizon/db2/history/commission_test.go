package history

import (
	"bitbucket.org/atticlab/go-smart-base/amount"
	"bitbucket.org/atticlab/go-smart-base/keypair"
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/assets"
	"bitbucket.org/atticlab/horizon/log"
	"bitbucket.org/atticlab/horizon/resource/base"
	"bitbucket.org/atticlab/horizon/test"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCommissionHash(t *testing.T) {
	log.DefaultLogger.Entry.Logger.Level = log.DebugLevel
	issuer, err := keypair.Random()
	assert.Nil(t, err)
	EUR := base.Asset{
		Type:   assets.MustString(xdr.AssetTypeAssetTypeCreditAlphanum4),
		Code:   "EUR",
		Issuer: issuer.Address(),
	}
	USD := EUR
	USD.Code = "USD"
	assert.NotEqual(t, EUR, USD)
	defaultFee := CommissionKey{}
	others := []CommissionKey{defaultFee}
	// only asset
	comKeyEUR := CommissionKey{
		Asset: EUR,
	}
	checkGreater(t, comKeyEUR, others)
	{
		comKeyUSD := CommissionKey{
			Asset: USD,
		}
		assert.Equal(t, 0, comKeyEUR.Compare(&comKeyUSD))
	}
	others = append(others, comKeyEUR)
	// only account type
	fromTypeKey := CommissionKey{
		FromType: 3,
	}
	checkGreater(t, fromTypeKey, others)
	{
		toTypeKey := CommissionKey{
			ToType: 1,
		}
		assert.Equal(t, 0, fromTypeKey.Compare(&toTypeKey))
	}
	others = append(others, fromTypeKey)
	// accountType and asset
	fromTypeAssetKey := fromTypeKey
	fromTypeAssetKey.Asset = EUR
	checkGreater(t, fromTypeAssetKey, others)
	others = append(others, fromTypeAssetKey)
	// only account
	accountId := issuer.Address()
	fromAcc := CommissionKey{
		From: accountId,
	}
	checkGreater(t, fromAcc, others)
	{
		toAcc := CommissionKey{
			To: accountId,
		}
		assert.Equal(t, 0, fromAcc.Compare(&toAcc))
	}
	others = append(others, fromAcc)
}

func checkGreater(t *testing.T, key CommissionKey, others []CommissionKey) {
	for _, other := range others {
		assert.Equal(t, 1, key.Compare(&other))
	}
}

func TestCommissionStore(t *testing.T) {
	log.DefaultLogger.Entry.Logger.Level = log.DebugLevel
	q := &Q{test.Start(t).HorizonRepo()}
	Convey("not exist", t, func() {
		keys := CreateCommissionKeys("from", "to", 1, 3, base.Asset{})
		commissions, err := q.CommissionByKey(keys)
		assert.Nil(t, err)
		assert.Equal(t, 0, len(commissions))
	})
	Convey("get", t, func() {
		var key CommissionKey
		{
			account, err := keypair.Random()
			assert.Nil(t, err)
			accountId := account.Address()
			var accountType int32
			key = CommissionKey{
				From:   accountId,
				ToType: accountType,
			}
		}
		commission, err := NewCommission(key, 10*amount.One, 12*amount.One)
		assert.Nil(t, err)
		err = q.InsertCommission(commission)
		assert.Nil(t, err)

		keys := CreateCommissionKeys(key.From, "to", 123, key.ToType, base.Asset{
			Type:   "random_type",
			Issuer: "random_issuer",
			Code:   "ASD",
		})
		stored, err := q.CommissionByKey(keys)
		assert.Nil(t, err)
		log.WithField("stored", stored).Debug("Got commission")
		assert.Equal(t, 1, len(stored))
		commission.Id = stored[0].Id
		stored[0].weight = commission.weight
		assert.Equal(t, *commission, stored[0])
		err = q.deleteCommissions()
		assert.Nil(t, err)
	})
	Convey("create keys", t, func() {
		keys := CreateCommissionKeys("from", "to", 1, 2, base.Asset{Type: "asset_type", Issuer: "Issuer", Code: "Code"})
		assert.Equal(t, 32, len(keys))
		for _, value := range keys {
			log.WithField("value", value).WithField("weight", value.CountWeight()).Info("got key")
		}
	})
	Convey("filter", t, func() {
		rawCommissions := []Commission {
		}
		filtered := filterByWeight(rawCommissions)
		assert.Equal(t, 0, len(filtered))
		rawCommissions = []Commission{
			Commission{
				weight: 2,
			},
			Commission{
				weight: 3,
			},
			Commission{
				weight: 3,
			},
		}
		filtered = filterByWeight(rawCommissions)
		assert.Equal(t, 2, len(filtered))
	})
}
