package transactions

import (
	"bitbucket.org/atticlab/go-smart-base/build"
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/db2/core"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/log"
	"bitbucket.org/atticlab/horizon/test"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	"time"
	"bitbucket.org/atticlab/horizon/cache"
)

func TestChangeTrustOpFrame(t *testing.T) {
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

	manager := NewManager(coreQ, historyQ, nil, &config, &cache.SharedCache{
		AccountHistoryCache: cache.NewHistoryAccount(historyQ),
	})

	Convey("Invalid asset", t, func() {
		changeTrust := build.ChangeTrust(build.Asset{
			Code:   "USD",
			Issuer: root.Address(),
		})
		tx := build.Transaction(changeTrust, build.Sequence{1}, build.SourceAccount{root.Address()})
		txE := NewTransactionFrame(&EnvelopeInfo{
			Tx: tx.Sign(root.Seed()).E,
		})
		opFrame := NewOperationFrame(&txE.Tx.Tx.Operations[0], txE, 1, time.Now())
		isValid, err := opFrame.CheckValid(manager)
		So(err, ShouldBeNil)
		So(isValid, ShouldBeFalse)
		So(opFrame.GetResult().Result.MustTr().MustChangeTrustResult().Code, ShouldEqual, xdr.ChangeTrustResultCodeChangeTrustMalformed)
		So(opFrame.GetResult().Info.GetError(), ShouldEqual, ASSET_NOT_ALLOWED.Error())
	})
	Convey("Success", t, func() {
		changeTrust := build.ChangeTrust(build.Asset{
			Code:   "UAH",
			Issuer: root.Address(),
		})
		tx := build.Transaction(changeTrust, build.Sequence{1}, build.SourceAccount{root.Address()})
		txE := NewTransactionFrame(&EnvelopeInfo{
			Tx: tx.Sign(root.Seed()).E,
		})
		opFrame := NewOperationFrame(&txE.Tx.Tx.Operations[0], txE, 1, time.Now())
		isValid, err := opFrame.CheckValid(manager)
		So(err, ShouldBeNil)
		So(isValid, ShouldBeTrue)
		So(opFrame.GetResult().Result.MustTr().MustChangeTrustResult().Code, ShouldEqual, xdr.ChangeTrustResultCodeChangeTrustSuccess)
	})
}
