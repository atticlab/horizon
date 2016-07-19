package validator

import (
	"bitbucket.org/atticlab/go-smart-base/build"
	"bitbucket.org/atticlab/go-smart-base/keypair"
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/admin"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/log"
	"bitbucket.org/atticlab/horizon/test"
	"encoding/json"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAdminActionProvider(t *testing.T) {
	tt := test.Start(t).Scenario("base")
	defer tt.Finish()

	log.DefaultLogger.Entry.Logger.Level = log.DebugLevel
	historyQ := &history.Q{tt.HorizonRepo()}

	Convey("Set commission Actions:", t, func() {
		signer, err := keypair.Random()
		So(err, ShouldBeNil)
		Convey("Several non admin operations", func() {
			newAccount, err := keypair.Random()
			So(err, ShouldBeNil)
			payment := build.Payment(build.CreditAmount{"USD", newAccount.Address(), "50.0"},
				build.Destination{newAccount.Address()})
			createAccount := build.CreateAccount(build.Destination{newAccount.Address()})
			tx := build.Transaction(payment, createAccount, build.Sequence{1}, build.SourceAccount{signer.Address()})
			txE := tx.Sign(signer.Seed())
			rawTxE, err := txE.Base64()
			So(err, ShouldBeNil)
			var newTxE xdr.TransactionEnvelope
			err = xdr.SafeUnmarshalBase64(rawTxE, &newTxE)
			So(err, ShouldBeNil)
			adminVal := NewAdministrativeValidator(historyQ)
			res, err := adminVal.CheckTransaction(&newTxE)
			So(err, ShouldBeNil)
			So(res, ShouldBeNil)
			for _, op := range newTxE.Tx.Operations {
				opResult, additionalInfo, err := adminVal.CheckOperation(newTxE.Tx.SourceAccount.Address(), &op)
				So(err, ShouldBeNil)
				So(additionalInfo, ShouldBeNil)
				opInner := opResult.MustTr()
				switch opInner.Type {
				case xdr.OperationTypePayment:
					res := opInner.MustPaymentResult()
					So(res.Code, ShouldEqual, xdr.PaymentResultCodePaymentSuccess)
				case xdr.OperationTypeCreateAccount:
					res := opInner.MustCreateAccountResult()
					So(res.Code, ShouldEqual, xdr.CreateAccountResultCodeCreateAccountSuccess)
				default:
					assert.Fail(t, "Invalid type")
				}

			}
		})
		Convey("Admin op", func() {
			opObjData := map[string]interface{}{
				string(admin.SubjectCommission): map[string]interface{}{},
			}
			opData, err := json.Marshal(opObjData)
			So(err, ShouldBeNil)
			adminOp := build.AdministrativeOp(build.OpLongData{OpData: string(opData)})
			So(adminOp.Err, ShouldBeNil)
			tx := build.Transaction(adminOp, build.Sequence{1}, build.SourceAccount{signer.Address()})
			So(tx.Err, ShouldBeNil)
			txE := tx.Sign(signer.Seed())
			rawTxE, err := txE.Base64()
			So(err, ShouldBeNil)
			var newTxE xdr.TransactionEnvelope
			err = xdr.SafeUnmarshalBase64(rawTxE, &newTxE)
			So(err, ShouldBeNil)
			adminVal := NewAdministrativeValidator(historyQ)
			res, err := adminVal.CheckTransaction(&newTxE)
			So(err, ShouldBeNil)
			So(res, ShouldBeNil)
			opResult, additionalInfo, err := adminVal.CheckOperation(newTxE.Tx.SourceAccount.Address(), &newTxE.Tx.Operations[0])
			So(err, ShouldBeNil)
			So(additionalInfo, ShouldBeNil)
			So(opResult.MustTr().MustAdminResult().Code, ShouldEqual, xdr.AdministrativeResultCodeAdministrativeSuccess)

		})
	})
}
