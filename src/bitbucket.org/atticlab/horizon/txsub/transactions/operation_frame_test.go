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

func TestOperationFrame(t *testing.T) {
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

	Convey("Test OperationFrame Frame:", t, func() {
		Convey("Source account does to exists", func() {
			createAccount := build.CreateAccount(build.Destination{newAccount.Address()})
			tx := build.Transaction(createAccount, build.Sequence{1}, build.SourceAccount{newAccount.Address()})
			txE := tx.Sign(newAccount.Seed()).E
			opFrame := NewOperationFrame(&createAccount.O, txE)
			isValid, err := opFrame.CheckValid(historyQ, coreQ, &config)
			So(err, ShouldBeNil)
			So(isValid, ShouldBeFalse)
			So(opFrame.GetResult().Result.Code, ShouldEqual, xdr.OperationResultCodeOpNoAccount)
		})
		Convey("Op source does not exists", func() {
			createAccount := build.CreateAccount(build.Destination{newAccount.Address()}, build.SourceAccount{newAccount.Address()})
			tx := build.Transaction(createAccount, build.Sequence{1}, build.SourceAccount{root.Address()})
			txE := tx.Sign(root.Seed()).E
			opFrame := NewOperationFrame(&createAccount.O, txE)
			isValid, err := opFrame.CheckValid(historyQ, coreQ, &config)
			So(err, ShouldBeNil)
			So(isValid, ShouldBeFalse)
			So(opFrame.GetResult().Result.Code, ShouldEqual, xdr.OperationResultCodeOpNoAccount)
		})
		Convey("Invalid op", func() {
			invalidOp := build.CreateAccount(build.Destination{newAccount.Address()}, build.SourceAccount{root.Address()})
			invalidOp.O.Body.Type = xdr.OperationType(123)
			tx := build.Transaction(invalidOp, build.Sequence{1}, build.SourceAccount{root.Address()})
			txE := tx.Sign(root.Seed()).E
			opFrame := NewOperationFrame(&invalidOp.O, txE)
			isValid, err := opFrame.CheckValid(historyQ, coreQ, &config)
			So(err.Error(), ShouldEqual, "unknown operation")
			So(isValid, ShouldBeFalse)
		})
	})
}
