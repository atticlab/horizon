package transactions

import (
	"github.com/atticlab/go-smart-base/build"
	"github.com/atticlab/go-smart-base/keypair"
	"github.com/atticlab/go-smart-base/xdr"
	"github.com/atticlab/horizon/cache"
	"github.com/atticlab/horizon/db2/core"
	"github.com/atticlab/horizon/db2/history"
	"github.com/atticlab/horizon/log"
	"github.com/atticlab/horizon/test"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestAccountMergeOpFrame(t *testing.T) {
	tt := test.Start(t).Scenario("base")
	defer tt.Finish()

	log.DefaultLogger.Entry.Logger.Level = log.DebugLevel

	root := test.BankMasterSeed()
	log.Info(root.Address())
	newAccount, err := keypair.Random()
	assert.Nil(t, err)

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

	Convey("Test Account Merge Op Frame:", t, func() {
		accountMerge := build.AccountMerge(build.Destination{newAccount.Address()})
		tx := build.Transaction(accountMerge, build.Sequence{1}, build.SourceAccount{root.Address()})
		txE := NewTransactionFrame(&EnvelopeInfo{
			Tx: tx.Sign(root.Seed()).E,
		})
		opFrame := NewOperationFrame(&txE.Tx.Tx.Operations[0], txE, 1, time.Now())
		isValid, err := opFrame.CheckValid(manager)
		So(err, ShouldBeNil)
		log.WithField("result", opFrame.Result).Info("Got result")
		So(isValid, ShouldBeTrue)
		So(opFrame.GetResult().Result.MustTr().MustAccountMergeResult().Code, ShouldEqual, xdr.AccountMergeResultCodeAccountMergeSuccess)
	})
}
