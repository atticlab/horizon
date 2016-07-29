package transactions

import (
	"bitbucket.org/atticlab/go-smart-base/build"
	"bitbucket.org/atticlab/go-smart-base/keypair"
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/db2/core"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/log"
	"bitbucket.org/atticlab/horizon/test"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	"github.com/stretchr/testify/assert"
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

	Convey("Invalid asset", t, func() {
		allowTrust := build.AllowTrust(build.AllowTrustAsset{"USD"}, build.Trustor{newAccount.Address()})
		tx := build.Transaction(allowTrust, build.Sequence{1}, build.SourceAccount{root.Address()})
		txE := tx.Sign(root.Seed()).E
		opFrame := NewOperationFrame(&txE.Tx.Operations[0], txE)
		isValid, err := opFrame.CheckValid(historyQ, coreQ, &config)
		So(err, ShouldBeNil)
		So(isValid, ShouldBeFalse)
		So(opFrame.GetResult().Result.MustTr().MustAllowTrustResult().Code, ShouldEqual, xdr.AllowTrustResultCodeAllowTrustMalformed)
		So(opFrame.GetResult().Info.GetError(), ShouldEqual, ASSET_NOT_ALLOWED.Error())
	})
	Convey("Success", t, func() {
		allowTrust := build.AllowTrust(build.AllowTrustAsset{"UAH"}, build.Trustor{root.Address()})
		tx := build.Transaction(allowTrust, build.Sequence{1}, build.SourceAccount{root.Address()})
		txE := tx.Sign(root.Seed()).E
		opFrame := NewOperationFrame(&txE.Tx.Operations[0], txE)
		isValid, err := opFrame.CheckValid(historyQ, coreQ, &config)
		So(err, ShouldBeNil)
		So(isValid, ShouldBeTrue)
		So(opFrame.GetResult().Result.MustTr().MustAllowTrustResult().Code, ShouldEqual, xdr.AllowTrustResultCodeAllowTrustSuccess)
	})
}
