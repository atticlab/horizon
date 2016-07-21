package validator

import (
	"bitbucket.org/atticlab/go-smart-base/build"
	"bitbucket.org/atticlab/go-smart-base/keypair"
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/log"
	"bitbucket.org/atticlab/horizon/test"
	"bitbucket.org/atticlab/horizon/txsub/results"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestAssetsValidator(t *testing.T) {
	tt := test.Start(t).Scenario("base")
	defer tt.Finish()

	log.DefaultLogger.Entry.Logger.Level = log.DebugLevel
	historyQ := &history.Q{tt.HorizonRepo()}
	conf := test.NewTestConfig()
	validator := NewAssetsValidator(historyQ, &conf)

	Convey("Operation without assets:", t, func() {
		newAccount, err := keypair.Random()
		So(err, ShouldBeNil)
		uah := build.Asset{
			Code:   "UAH",
			Issuer: conf.BankMasterKey,
			Native: false,
		}
		checkAssetValid(validator, uah)
		auah := build.Asset{
			Code:   "AUAH",
			Issuer: conf.BankMasterKey,
			Native: false,
		}
		checkAssetValid(validator, auah)
		Convey("CreateAccount", func() {
			op := build.CreateAccount(build.Destination{newAccount.Address()})
			successCheckTx(validator, newAccount, op)
		})
		Convey("SetOptions", func() {
			op := build.SetOptions(build.HomeDomain("random_domain"))
			successCheckTx(validator, newAccount, op)
		})
		Convey("AccountMerge", func() {
			op := build.AccountMerge(build.Destination{newAccount.Address()})
			successCheckTx(validator, newAccount, op)
		})
		Convey("Inflation", func() {
			op := build.Inflation()
			successCheckTx(validator, newAccount, op)
		})
		Convey("ManageData", func() {
			op := build.ClearData("random_name")
			successCheckTx(validator, newAccount, op)
		})
		Convey("AdminOp", func() {
			op := build.AdministrativeOp(build.OpLongData{"random_name"})
			successCheckTx(validator, newAccount, op)
		})
		Convey("Not allowed asset", func() {
			notAllowed := "USD"
			notAllowedAsset := build.Asset{
				Code:   notAllowed,
				Issuer: conf.BankMasterKey,
			}
			Convey("payment", func() {
				payment := build.Payment(build.CreditAmount{notAllowed, conf.BankMasterKey, "50.0"},
					build.Destination{conf.BankMasterKey})
				checkAssetNotSupported(validator, newAccount, payment)
			})
			Convey("pathpayment", func() {
				checkAssetValid(validator, auah)
				// source asset invalid
				payment := build.Payment(build.PayWithPath{
					Asset:     notAllowedAsset,
					MaxAmount: "1000",
				},
					build.Destination{conf.BankMasterKey})
				checkAssetNotSupported(validator, newAccount, payment)
				// asset in path is invalid
				payment = build.Payment(build.PayWithPath{
					Asset:     uah,
					MaxAmount: "1000",
					Path: []build.Asset{
						notAllowedAsset,
						auah,
					},
				},
					build.Destination{conf.BankMasterKey})
				checkAssetNotSupported(validator, newAccount, payment)
				// dest asset is invalid
				payment = build.Payment(build.PayWithPath{
					Asset:     uah,
					MaxAmount: "1000",
					Path: []build.Asset{
						auah,
						notAllowedAsset,
					},
				},
					build.Destination{conf.BankMasterKey})
				checkAssetNotSupported(validator, newAccount, payment)

			})
			Convey("manageOffer", func() {
				// selling is invalid
				offer := build.ManageOffer(false, build.Rate{
					Selling: notAllowedAsset,
					Buying:  uah,
					Price:   build.Price("10.1"),
				})
				checkAssetNotSupported(validator, newAccount, offer)
				// buying is invalid
				offer = build.ManageOffer(false, build.Rate{
					Selling: uah,
					Buying:  notAllowedAsset,
					Price:   build.Price("10.1"),
				})
				checkAssetNotSupported(validator, newAccount, offer)
			})
			Convey("passiveOffer", func() {
				// selling is invalid
				offer := build.ManageOffer(true, build.Rate{
					Selling: notAllowedAsset,
					Buying:  uah,
					Price:   build.Price("10.1"),
				})
				checkAssetNotSupported(validator, newAccount, offer)
				// buying is invalid
				offer = build.ManageOffer(true, build.Rate{
					Selling: uah,
					Buying:  notAllowedAsset,
					Price:   build.Price("10.1"),
				})
				checkAssetNotSupported(validator, newAccount, offer)
			})
			Convey("changeTrust", func() {
				changeTrust := build.ChangeTrust(notAllowedAsset)
				checkAssetNotSupported(validator, newAccount, changeTrust)
			})
			Convey("allowTrust", func() {
				allowTrust := build.AllowTrust(build.Authorize{true}, build.AllowTrustAsset{"AUAH"}, build.Trustor{newAccount.Address()})
				checkAssetNotSupported(validator, newAccount, allowTrust)
			})
		})
		Convey("Allowed asset", func() {
			Convey("payment", func() {
				// anon asset
				payment := build.Payment(build.CreditAmount{"AUAH", conf.BankMasterKey, "50.0"},
					build.Destination{newAccount.Address()})
				successCheckTx(validator, newAccount, payment)
				// non anon - account exists
				payment = build.Payment(build.CreditAmount{"UAH", conf.BankMasterKey, "50.0"},
					build.Destination{conf.BankMasterKey})
				successCheckTx(validator, newAccount, payment)
				// non anon - dest not found
				payment = build.Payment(build.CreditAmount{"UAH", conf.BankMasterKey, "50.0"},
					build.Destination{newAccount.Address()})
				checkDestNotFound(validator, newAccount, payment)
			})
			Convey("pathpayment", func() {
				// anon asset
				payment := build.Payment(
					build.PayWithPath{
						Asset:     uah,
						MaxAmount: "1000",
					},
					build.Destination{
						newAccount.Address(),
					},
					build.CreditAmount{
						Code:   auah.Code,
						Issuer: auah.Issuer,
						Amount: "1000",
					})
				successCheckTx(validator, newAccount, payment)
				// non anon - account exists
				payment = build.Payment(
					build.PayWithPath{
						Asset:     uah,
						MaxAmount: "1000",
					},
					build.Destination{
						conf.BankMasterKey,
					},
					build.CreditAmount{
						Code:   uah.Code,
						Issuer: uah.Issuer,
						Amount: "1000",
					})
				successCheckTx(validator, newAccount, payment)
				// non anon - dest not found
				payment = build.Payment(
					build.PayWithPath{
						Asset:     auah,
						MaxAmount: "1000",
					},
					build.Destination{
						newAccount.Address(),
					},
					build.CreditAmount{
						Code:   uah.Code,
						Issuer: uah.Issuer,
						Amount: "1000",
					})
				checkDestNotFound(validator, newAccount, payment)

			})
			/*Convey("manageOffer", func() {
				offer := build.ManageOffer(false, build.Rate{
					Selling: auah,
					Buying: uah,
				})
				successCheckTx(validator, newAccount, offer)
			})
			Convey("passiveOffer", func() {
				// selling is invalid
				offer := build.ManageOffer(true, build.Rate{
					Selling: auah,
					Buying: uah,
				})
				successCheckTx(validator, newAccount, offer)
			})
			Convey("changeTrust", func() {
				changeTrust := build.ChangeTrust(auah)
				successCheckTx(validator, newAccount, changeTrust)
			})*/
		})
	})
}

func checkAssetValid(v *AssetsValidator, asset build.Asset) {
	xdrAsset, err := asset.ToXdrObject()
	So(err, ShouldBeNil)
	isValid, err := v.isAssetValid(xdrAsset)
	So(err, ShouldBeNil)
	So(isValid, ShouldBeTrue)
}

func checkDestNotFound(v ValidatorInterface, account *keypair.Full, ops ...build.TransactionMutator) {
	newTxE := createTx(account, ops...)
	result, err := v.CheckTransaction(newTxE)
	So(err, ShouldBeNil)
	So(result, ShouldBeNil)
	for _, op := range newTxE.Tx.Operations {
		opResult, addInfo, err := v.CheckOperation(newTxE.Tx.SourceAccount, &op)
		So(err, ShouldBeNil)
		So(addInfo, ShouldBeNil)
		switch op.Body.Type {
		case xdr.OperationTypePayment:
			So(opResult.MustTr().MustPaymentResult().Code, ShouldEqual, xdr.PaymentResultCodePaymentNoDestination)
		case xdr.OperationTypePathPayment:
			So(opResult.MustTr().MustPathPaymentResult().Code, ShouldEqual, xdr.PathPaymentResultCodePathPaymentNoDestination)
		}
	}
}

func checkAssetNotSupported(v ValidatorInterface, account *keypair.Full, ops ...build.TransactionMutator) {
	newTxE := createTx(account, ops...)
	result, err := v.CheckTransaction(newTxE)
	So(err, ShouldBeNil)
	So(result, ShouldBeNil)
	for _, op := range newTxE.Tx.Operations {
		opResult, addInfo, err := v.CheckOperation(newTxE.Tx.SourceAccount, &op)
		So(err, ShouldBeNil)
		So(addInfo["error"], ShouldEqual, "asset_not_suppoted")
		isSuccessful, err := results.IsSuccessful(opResult)
		So(err, ShouldBeNil)
		So(isSuccessful, ShouldBeFalse)
	}
}

func createTx(account *keypair.Full, ops ...build.TransactionMutator) *xdr.TransactionEnvelope {
	tx := build.Transaction(build.Sequence{1}, build.SourceAccount{AddressOrSeed: account.Address()})
	tx.Mutate(ops...)
	txE := tx.Sign(account.Seed())
	rawTxE, err := txE.Base64()
	So(err, ShouldBeNil)
	var newTxE xdr.TransactionEnvelope
	err = xdr.SafeUnmarshalBase64(rawTxE, &newTxE)
	So(err, ShouldBeNil)
	return &newTxE
}

func successCheckTx(v ValidatorInterface, account *keypair.Full, ops ...build.TransactionMutator) {
	newTxE := createTx(account, ops...)
	result, err := v.CheckTransaction(newTxE)
	So(err, ShouldBeNil)
	So(result, ShouldBeNil)
	for _, op := range newTxE.Tx.Operations {
		opResult, addInfo, err := v.CheckOperation(newTxE.Tx.SourceAccount, &op)
		So(err, ShouldBeNil)
		So(addInfo, ShouldBeNil)
		isSuccessful, err := results.IsSuccessful(opResult)
		So(err, ShouldBeNil)
		So(isSuccessful, ShouldBeTrue)
	}
}
