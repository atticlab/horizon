package transactions

import (
	"github.com/atticlab/go-smart-base/build"
	"github.com/atticlab/go-smart-base/keypair"
	"github.com/atticlab/go-smart-base/xdr"
	"github.com/atticlab/horizon/db2/core"
	"github.com/atticlab/horizon/db2/history"
	"github.com/atticlab/horizon/log"
	"github.com/atticlab/horizon/test"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	"github.com/stretchr/testify/assert"
	"time"
	"github.com/atticlab/horizon/cache"
)

func TestAllowTrustOpFrame(t *testing.T) {
	tt := test.Start(t).Scenario("base")
	defer tt.Finish()

	log.DefaultLogger.Entry.Logger.Level = log.DebugLevel

	historyQ := &history.Q{
		tt.HorizonRepo(),
	}
	coreQ := &core.Q{
		tt.CoreRepo(),
	}
	config := test.NewTestConfig()

	root := test.BankMasterSeed()
	newAccount, err := keypair.Random()
	assert.Nil(t, err)

	manager := NewManager(coreQ, historyQ, nil, &config, &cache.SharedCache{
		AccountHistoryCache: cache.NewHistoryAccount(historyQ),
	})

	Convey("Invalid asset", t, func() {
		allowTrust := build.AllowTrust(build.AllowTrustAsset{"USD"}, build.Trustor{newAccount.Address()})
		tx := build.Transaction(allowTrust, build.Sequence{1}, build.SourceAccount{root.Address()})
		txE := NewTransactionFrame(&EnvelopeInfo{
			Tx: tx.Sign(root.Seed()).E,
		})
		opFrame := NewOperationFrame(&txE.Tx.Tx.Operations[0], txE, 1, time.Now())
		isValid, err := opFrame.CheckValid(manager)
		So(err, ShouldBeNil)
		So(isValid, ShouldBeFalse)
		So(opFrame.GetResult().Result.MustTr().MustAllowTrustResult().Code, ShouldEqual, xdr.AllowTrustResultCodeAllowTrustMalformed)
		So(opFrame.GetResult().Info.GetError(), ShouldEqual, ASSET_NOT_ALLOWED.Error())
	})
	Convey("Success", t, func() {
		allowTrust := build.AllowTrust(build.AllowTrustAsset{"UAH"}, build.Trustor{root.Address()})
		tx := build.Transaction(allowTrust, build.Sequence{1}, build.SourceAccount{root.Address()})
		txE := NewTransactionFrame(&EnvelopeInfo{
			Tx: tx.Sign(root.Seed()).E,
		})
		opFrame := NewOperationFrame(&txE.Tx.Tx.Operations[0], txE, 1, time.Now())
		isValid, err := opFrame.CheckValid(manager)
		So(err, ShouldBeNil)
		So(isValid, ShouldBeTrue)
		So(opFrame.GetResult().Result.MustTr().MustAllowTrustResult().Code, ShouldEqual, xdr.AllowTrustResultCodeAllowTrustSuccess)
	})
}
