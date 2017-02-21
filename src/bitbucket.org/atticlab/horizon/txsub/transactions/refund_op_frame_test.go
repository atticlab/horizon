package transactions

import (
	"bitbucket.org/atticlab/go-smart-base/build"
	"bitbucket.org/atticlab/go-smart-base/keypair"
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/cache"
	"bitbucket.org/atticlab/horizon/db2/core"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/db2/history/details"
	"bitbucket.org/atticlab/horizon/log"
	"bitbucket.org/atticlab/horizon/test"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/guregu/null"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"math/rand"
	"testing"
	"time"
)

func TestRefundOpFrame(t *testing.T) {
	log.DefaultLogger.Entry.Logger.Level = log.DebugLevel

	historyQ := history.QMock{}
	coreQ := core.QMock{}
	config := test.NewTestConfig()

	root := test.BankMasterSeed()

	manager := NewManager(&coreQ, &historyQ, nil, &config, &cache.SharedCache{
		AccountHistoryCache: cache.NewHistoryAccount(&historyQ),
	})

	paymentSenderKP, err := keypair.Random()
	assert.Nil(t, err)

	paymentID := rand.Int63()
	paymentAmount := "125.78"
	commissionAmount := "22.34"

	assetCode := "USD"
	paymentRefund := build.Refund(build.CreditAmount{
		Code:   assetCode,
		Issuer: root.Address(),
		Amount: paymentAmount,
	}, build.OriginalAmount{
		Amount: paymentAmount,
	}, build.PaymentID{
		ID: paymentID,
	}, build.PaymentSender{
		AddressOrSeed: paymentSenderKP.Address(),
	})

	tx := build.Transaction(paymentRefund, build.Sequence{1}, build.SourceAccount{root.Address()})
	txE := NewTransactionFrame(&EnvelopeInfo{
		Tx: tx.Sign(root.Seed()).E,
	})

	validOperation := txE.Tx.Tx.Operations[0]

	now := time.Now()

	historyQ.On("AccountByAddress", root.Address()).Return(history.Account{
		Address: root.Address(),
	}, nil)
	Convey("Negative amount", t, func() {
		operation := validOperation
		refundOp := *operation.Body.RefundOp
		operation.Body.RefundOp = &refundOp
		operation.Body.RefundOp.Amount = xdr.Int64(-100)
		opFrame := NewOperationFrame(&operation, txE, 1, now)
		isValid, err := opFrame.CheckValid(manager)
		So(err, ShouldBeNil)
		So(isValid, ShouldBeFalse)
		So(opFrame.GetResult().Result.MustTr().MustRefundResult().Code, ShouldEqual, xdr.RefundResultCodeRefundMalformed)
	})
	Convey("Given valid payment reversal op", t, func() {
		log.Error("Given valid payment reversal op")
		operation := validOperation
		opFrame := NewOperationFrame(&operation, txE, 1, now)
		Convey("Failed to get payment", func() {
			expectedError := errors.New("Failed to get payment from db")
			historyQ.On("OperationByID", mock.Anything, paymentID).Return(expectedError).Once()
			isValid, err := opFrame.CheckValid(manager)
			So(err, ShouldNotBeNil)
			So(expectedError.Error(), ShouldEqual, err.Error())
			So(isValid, ShouldBeFalse)
		})
		Convey("Payment does not exists", func() {
			historyQ.On("OperationByID", mock.Anything, paymentID).Return(sql.ErrNoRows).Once()
			isValid, err := opFrame.CheckValid(manager)
			So(err, ShouldBeNil)
			So(isValid, ShouldBeFalse)
			So(opFrame.GetResult().Result.MustTr().MustRefundResult().Code, ShouldEqual, xdr.RefundResultCodeRefundPaymentDoesNotExists)
		})
		Convey("Operation with same ID, but not payment", func() {
			historyQ.On("OperationByID", mock.Anything, paymentID).Run(func(args mock.Arguments) {
				op := args.Get(0).(*history.Operation)
				op.Type = xdr.OperationTypeAllowTrust
			}).Return(nil).Once()
			isValid, err := opFrame.CheckValid(manager)
			So(err, ShouldBeNil)
			So(isValid, ShouldBeFalse)
			So(opFrame.GetResult().Result.MustTr().MustRefundResult().Code, ShouldEqual, xdr.RefundResultCodeRefundPaymentDoesNotExists)
		})
		Convey("Given valid stored payment", func() {
			validStoredPayment := history.Operation{
				Type:          xdr.OperationTypePayment,
				ClosedAt:      now,
				SourceAccount: operation.Body.RefundOp.PaymentSource.Address(),
			}

			validPaymentDetails := details.Payment{
				From:   validStoredPayment.SourceAccount,
				To:     root.Address(),
				Amount: paymentAmount,
				Asset: details.Asset{
					Type:   "credit_alphanum4",
					Code:   assetCode,
					Issuer: root.Address(),
				},
				Fee: details.Fee{
					AmountCharged: &commissionAmount,
				},
			}

			jsonDetails, err := json.Marshal(validPaymentDetails)
			assert.Nil(t, err)
			validStoredPayment.DetailsString = null.StringFrom(string(jsonDetails))
			Convey("Valid max reversal duration", func() {
				opChecker := func(storedPayment history.Operation, expectedCode xdr.RefundResultCode) {
					historyQ.On("OperationByID", mock.Anything, paymentID).Run(func(args mock.Arguments) {
						op := args.Get(0).(*history.Operation)
						*op = storedPayment
					}).Return(nil).Once()
					isValid, err := opFrame.CheckValid(manager)
					So(err, ShouldBeNil)
					So(isValid, ShouldBeFalse)
					So(opFrame.GetResult().Result.MustTr().MustRefundResult().Code, ShouldEqual, expectedCode)
				}
				Convey("Invalid payment reversal source", func() {
					storedPayment := validStoredPayment
					storedPayment.SourceAccount = root.Address()
					opChecker(storedPayment, xdr.RefundResultCodeRefundInvalidSource)
				})
				Convey("Invalid payment details", func() {
					storedPayment := validStoredPayment
					storedPayment.DetailsString = null.StringFrom("")
					historyQ.On("OperationByID", mock.Anything, paymentID).Run(func(args mock.Arguments) {
						op := args.Get(0).(*history.Operation)
						*op = storedPayment
					}).Return(nil).Once()
					isValid, err := opFrame.CheckValid(manager)
					So(err, ShouldNotBeNil)
					So(isValid, ShouldBeFalse)
				})
				Convey("Invalid payment source", func() {
					storedPayment := validStoredPayment
					paymentDetails := validPaymentDetails
					paymentDetails.To = paymentSenderKP.Address()
					jsonDetails, err = json.Marshal(paymentDetails)
					assert.Nil(t, err)
					storedPayment.DetailsString = null.StringFrom(string(jsonDetails))
					opChecker(storedPayment, xdr.RefundResultCodeRefundInvalidPaymentSender)
				})
				Convey("Invalid amount", func() {
					storedPayment := validStoredPayment
					paymentDetails := validPaymentDetails
					paymentDetails.Amount = "102"
					jsonDetails, err = json.Marshal(paymentDetails)
					assert.Nil(t, err)
					storedPayment.DetailsString = null.StringFrom(string(jsonDetails))
					opChecker(storedPayment, xdr.RefundResultCodeRefundInvalidAmount)
				})
				Convey("Invalid asset", func() {
					storedPayment := validStoredPayment
					paymentDetails := validPaymentDetails
					Convey("Invalid asset code", func() {
						paymentDetails.Asset.Code = "EUR"
						jsonDetails, err = json.Marshal(paymentDetails)
						assert.Nil(t, err)
						storedPayment.DetailsString = null.StringFrom(string(jsonDetails))
						opChecker(storedPayment, xdr.RefundResultCodeRefundInvalidAsset)
					})
					Convey("Invalid asset issuer", func() {
						paymentDetails.Asset.Issuer = paymentSenderKP.Address()
						jsonDetails, err = json.Marshal(paymentDetails)
						assert.Nil(t, err)
						storedPayment.DetailsString = null.StringFrom(string(jsonDetails))
						opChecker(storedPayment, xdr.RefundResultCodeRefundInvalidAsset)
					})
					Convey("Invalid asset type", func() {
						paymentDetails.Asset.Type = "credit_alphanum12"
						jsonDetails, err = json.Marshal(paymentDetails)
						assert.Nil(t, err)
						storedPayment.DetailsString = null.StringFrom(string(jsonDetails))
						opChecker(storedPayment, xdr.RefundResultCodeRefundInvalidAsset)
					})

				})
				Convey("Success", func() {
					storedPayment := validStoredPayment
					historyQ.On("OperationByID", mock.Anything, paymentID).Run(func(args mock.Arguments) {
						op := args.Get(0).(*history.Operation)
						*op = storedPayment
					}).Return(nil).Once()
					isValid, err := opFrame.CheckValid(manager)
					So(err, ShouldBeNil)
					So(isValid, ShouldBeTrue)
					So(opFrame.GetResult().Result.MustTr().MustRefundResult().Code, ShouldEqual, xdr.RefundResultCodeRefundSuccess)
				})
			})
		})

	})
}
