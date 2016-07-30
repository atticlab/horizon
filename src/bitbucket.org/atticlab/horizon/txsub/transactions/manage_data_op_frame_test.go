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
)

func TestManageDataOpFrame(t *testing.T) {
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

	Convey("Success", t, func() {
		manageData := build.ClearData("random_name")
		tx := build.Transaction(manageData, build.Sequence{1}, build.SourceAccount{root.Address()})
		txE := tx.Sign(root.Seed()).E
		opFrame := NewOperationFrame(&txE.Tx.Operations[0], txE, time.Now())
		isValid, err := opFrame.CheckValid(historyQ, coreQ, &config)
		So(err, ShouldBeNil)
		So(isValid, ShouldBeTrue)
		So(opFrame.GetResult().Result.MustTr().MustManageDataResult().Code, ShouldEqual, xdr.ManageDataResultCodeManageDataSuccess)
	})
}
