package transactions

import (
	"github.com/atticlab/go-smart-base/build"
	"github.com/atticlab/go-smart-base/xdr"
	"github.com/atticlab/horizon/cache"
	"github.com/atticlab/horizon/db2/core"
	"github.com/atticlab/horizon/db2/history"
	"github.com/atticlab/horizon/log"
	"github.com/atticlab/horizon/test"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	"time"
)

func TestInflationOpFrame(t *testing.T) {
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

	manager := NewManager(coreQ, historyQ, nil, &config, &cache.SharedCache{
		AccountHistoryCache: cache.NewHistoryAccount(historyQ),
	})

	root := test.BankMasterSeed()

	Convey("Success", t, func() {
		inflation := build.Inflation(build.SourceAccount{root.Address()})
		tx := build.Transaction(inflation, build.Sequence{1}, build.SourceAccount{root.Address()})
		txE := NewTransactionFrame(&EnvelopeInfo{
			Tx: tx.Sign(root.Seed()).E,
		})
		opFrame := NewOperationFrame(&txE.Tx.Tx.Operations[0], txE, 1, time.Now())
		isValid, err := opFrame.CheckValid(manager)
		So(err, ShouldBeNil)
		So(isValid, ShouldBeTrue)
		So(opFrame.GetResult().Result.MustTr().MustInflationResult().Code, ShouldEqual, xdr.InflationResultCodeInflationSuccess)
	})
}
