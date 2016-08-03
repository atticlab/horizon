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
	"time"
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

	manager := NewManager(coreQ, historyQ, nil, &config)

	root := test.BankMasterSeed()
	newAccount, err := keypair.Random()
	assert.Nil(t, err)

	Convey("Test OperationFrame Frame:", t, func() {
		Convey("Source account does to exists", func() {
			createAccount := build.CreateAccount(build.Destination{newAccount.Address()})
			tx := build.Transaction(createAccount, build.Sequence{1}, build.SourceAccount{newAccount.Address()})
			txE := NewTransactionFrame(&EnvelopeInfo{
				Tx: tx.Sign(root.Seed()).E,
			})
			opFrame := NewOperationFrame(&txE.Tx.Tx.Operations[0], txE, 1, time.Now())
			isValid, err := opFrame.CheckValid(manager)
			So(err, ShouldBeNil)
			So(isValid, ShouldBeFalse)
			So(opFrame.GetResult().Result.Code, ShouldEqual, xdr.OperationResultCodeOpNoAccount)
		})
		Convey("Op source does not exists", func() {
			createAccount := build.CreateAccount(build.Destination{newAccount.Address()}, build.SourceAccount{newAccount.Address()})
			tx := build.Transaction(createAccount, build.Sequence{1}, build.SourceAccount{root.Address()})
			txE := NewTransactionFrame(&EnvelopeInfo{
				Tx: tx.Sign(root.Seed()).E,
			})
			opFrame := NewOperationFrame(&txE.Tx.Tx.Operations[0], txE, 1, time.Now())
			isValid, err := opFrame.CheckValid(manager)
			So(err, ShouldBeNil)
			So(isValid, ShouldBeFalse)
			So(opFrame.GetResult().Result.Code, ShouldEqual, xdr.OperationResultCodeOpNoAccount)
		})
	})
}
