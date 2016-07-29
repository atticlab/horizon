package transactions

import (
	"bitbucket.org/atticlab/go-smart-base/build"
	"bitbucket.org/atticlab/go-smart-base/keypair"
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/config"
	"bitbucket.org/atticlab/horizon/db2/core"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/log"
	"bitbucket.org/atticlab/horizon/test"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPathPaymentOpFrame(t *testing.T) {
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

	Convey("Invalid source asset", t, func() {
		payment := build.Payment(build.Destination{newAccount.Address()}, build.PayWithPath{
			Asset: build.Asset{
				Code:   "USD",
				Issuer: root.Address(),
			},
			MaxAmount: "1000000",
		})
		checkPaymentInvalidAsset(payment, root, historyQ, coreQ, config)
	})
	Convey("Invalid dest asset", t, func() {
		payment := build.Payment(build.Destination{newAccount.Address()}, build.PayWithPath{
			Asset: build.Asset{
				Code:   "UAH",
				Issuer: root.Address(),
			},
			Path: []build.Asset{
				build.Asset{
					Code: "USD",
					Issuer: root.Address(),
				},
			},
			MaxAmount: "1000000",
		})
		checkPaymentInvalidAsset(payment, root, historyQ, coreQ, config)
	})
	Convey("Invalid path asset", t, func() {
		payment := build.Payment(build.Destination{newAccount.Address()}, build.PayWithPath{
			Asset: build.Asset{
				Code:   "UAH",
				Issuer: root.Address(),
			},
			Path: []build.Asset{
				build.Asset{
					Code: "USD",
					Issuer: root.Address(),
				},
				build.Asset{
					Code: "AUAH",
					Issuer: root.Address(),
				},
			},
			MaxAmount: "1000000",
		})
		checkPaymentInvalidAsset(payment, root, historyQ, coreQ, config)
	})
}

func checkPaymentInvalidAsset(payment build.PaymentBuilder, root *keypair.Full, historyQ history.QInterface, coreQ core.QInterface, config config.Config) {
	tx := build.Transaction(payment, build.Sequence{1}, build.SourceAccount{root.Address()})
	txE := tx.Sign(root.Seed()).E
	opFrame := NewOperationFrame(&txE.Tx.Operations[0], txE)
	isValid, err := opFrame.CheckValid(historyQ, coreQ, &config)
	So(err, ShouldBeNil)
	So(isValid, ShouldBeFalse)
	So(opFrame.GetResult().Result.MustTr().MustPathPaymentResult().Code, ShouldEqual, xdr.PathPaymentResultCodePathPaymentMalformed)
	So(opFrame.GetResult().Info.GetError(), ShouldEqual, ASSET_NOT_ALLOWED.Error())
}
