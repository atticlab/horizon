package transactions

import (
	"bitbucket.org/atticlab/go-smart-base/build"
	"bitbucket.org/atticlab/go-smart-base/keypair"
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/db2/core"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/log"
	"bitbucket.org/atticlab/horizon/test"
	"bitbucket.org/atticlab/horizon/txsub/results"
	"bitbucket.org/atticlab/horizon/txsub/transactions/validators"
	"database/sql"
	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"github.com/go-errors/errors"
	"time"
)

func TestPaymentOpFrame(t *testing.T) {
	config := test.NewTestConfig()

	log.DefaultLogger.Entry.Logger.Level = log.DebugLevel

	bankMasterKey := test.BankMasterSeed()

	from := bankMasterKey
	to, err := keypair.Random()
	assert.Nil(t, err)
	historyQMock := &history.QMock{}
	destAsset := history.Asset{
		Type:   int(xdr.AssetTypeAssetTypeCreditAlphanum4),
		Code:   "UAH",
		Issuer: bankMasterKey.Address(),
	}
	fromType := xdr.AccountTypeAccountBank
	toType := xdr.AccountTypeAccountAnonymousUser
	coreQMock := &core.QMock{}
	coreQMock.On("AccountByAddress", from.Address()).Return(core.Account{
		Accountid:   from.Address(),
		AccountType: fromType,
	}, nil)
	payment := build.Payment(build.Destination{to.Address()}, build.CreditAmount{
		Code:   destAsset.Code,
		Issuer: destAsset.Issuer,
		Amount: "1000000",
	})
	tx := build.Transaction(payment, build.Sequence{1}, build.SourceAccount{from.Address()})
	txE := tx.Sign(bankMasterKey.Seed()).E
	opFrame := NewOperationFrame(&txE.Tx.Operations[0], txE, time.Now())
	paymentFrame := GetPaymentOpFrame(&opFrame)
	accountTypeVMock := &validators.AccountTypeValidatorMock{}
	paymentFrame.accountTypeValidator = accountTypeVMock
	assetVMock := &validators.AssetsValidatorMock{}
	paymentFrame.assetsValidator = assetVMock
	traitsMock := &validators.TraitsValidatorMock{}
	paymentFrame.traitsValidator = traitsMock
	outLimitsValidator := &validators.OutgoingLimitsValidatorMock{}
	paymentFrame.defaultOutLimitsValidator = outLimitsValidator
	inLimitsValidator := &validators.IncomingLimitsValidatorMock{}
	paymentFrame.defaultInLimitsValidator = inLimitsValidator
	Convey("Invalid asset", t, func() {
		assetVMock.On("GetValidAsset", mock.Anything).Return(nil, nil).Once()
		isValid, err := opFrame.CheckValid(historyQMock, coreQMock, &config)
		So(err, ShouldBeNil)
		So(isValid, ShouldBeFalse)
		So(opFrame.GetResult().Result.MustTr().MustPaymentResult().Code, ShouldEqual, xdr.PaymentResultCodePaymentMalformed)
		So(opFrame.GetResult().Info.GetError(), ShouldEqual, ASSET_NOT_ALLOWED.Error())
	})
	assetVMock.On("GetValidAsset", mock.Anything).Return(&destAsset, nil)
	Convey("Dest does not exists", t, func() {
		coreQMock.On("AccountByAddress", to.Address()).Return(nil, sql.ErrNoRows).Once()
		isValid, err := opFrame.CheckValid(historyQMock, coreQMock, &config)
		So(err, ShouldBeNil)
		So(isValid, ShouldBeFalse)
		So(opFrame.GetResult().Result.MustTr().MustPaymentResult().Code, ShouldEqual, xdr.PaymentResultCodePaymentNoDestination)
	})
	coreQMock.On("AccountByAddress", to.Address()).Return(core.Account{
		Accountid:   to.Address(),
		AccountType: toType,
	}, nil)
	Convey("Dest trust line does not exists", t, func() {
		coreQMock.On("TrustlineByAddressAndAsset", to.Address(), destAsset.Code, destAsset.Issuer).Return(nil, sql.ErrNoRows).Once()
		isValid, err := opFrame.CheckValid(historyQMock, coreQMock, &config)
		So(err, ShouldBeNil)
		So(isValid, ShouldBeFalse)
		So(opFrame.GetResult().Result.MustTr().MustPaymentResult().Code, ShouldEqual, xdr.PaymentResultCodePaymentNoTrust)
	})
	coreQMock.On("TrustlineByAddressAndAsset", to.Address(), destAsset.Code, destAsset.Issuer).Return(core.Trustline{
		Accountid: to.Address(),
		Balance:   xdr.Int64(0),
	}, nil)
	Convey("Account type restricted", t, func() {
		accountTypeVMock.On("VerifyAccountTypesForPayment", mock.Anything, mock.Anything).Return(&results.RestrictedForAccountTypeError{
			Reason: fmt.Sprintf("Payments from %s to %s are restricted.", fromType.String(), toType.String()),
		}).Once()
		isValid, err := opFrame.CheckValid(historyQMock, coreQMock, &config)
		So(err, ShouldBeNil)
		So(isValid, ShouldBeFalse)
		So(opFrame.GetResult().Result.MustTr().MustPaymentResult().Code, ShouldEqual, xdr.PaymentResultCodePaymentMalformed)
		So(opFrame.GetResult().Info.GetError(), ShouldEqual, fmt.Sprintf("Payments from %s to %s are restricted.", fromType.String(), toType.String()))
	})
	accountTypeVMock.On("VerifyAccountTypesForPayment", mock.Anything, mock.Anything).Return(nil)
	Convey("Failed to get traits", t, func() {
		errorData := "failed to get traits"
		traitsMock.On("CheckTraits", from.Address(), to.Address()).Return(nil, errors.New(errorData)).Once()
		isValid, err := opFrame.CheckValid(historyQMock, coreQMock, &config)
		So(err.Error(), ShouldEqual, errorData)
		So(isValid, ShouldBeFalse)
	})
	Convey("One of accounts - restricted", t, func() {
		errorData := "account_restricted"
		traitsMock.On("CheckTraits", from.Address(), to.Address()).Return(&results.RestrictedForAccountError{
			Reason: errorData,
		}, nil).Once()
		isValid, err := opFrame.CheckValid(historyQMock, coreQMock, &config)
		So(err, ShouldBeNil)
		So(isValid, ShouldBeFalse)
		So(opFrame.GetResult().Result.MustTr().MustPaymentResult().Code, ShouldEqual, xdr.PaymentResultCodePaymentMalformed)
		So(opFrame.GetResult().Info.GetError(), ShouldEqual, errorData)
	})
	traitsMock.On("CheckTraits", from.Address(), to.Address()).Return(nil, nil)
	Convey("Failed to validate out limits", t, func() {
		errorData := "limits_failed"
		outLimitsValidator.On("VerifyLimits").Return(nil, errors.New(errorData)).Once()
		isValid, err := opFrame.CheckValid(historyQMock, coreQMock, &config)
		So(err.Error(), ShouldEqual, errorData)
		So(isValid, ShouldBeFalse)
	})
	Convey("Outlimits exceeded", t, func() {
		errorData := "account_restricted"
		outLimitsValidator.On("VerifyLimits").Return(&results.ExceededLimitError{
			Description: errorData,
		}, nil).Once()
		isValid, err := opFrame.CheckValid(historyQMock, coreQMock, &config)
		So(err, ShouldBeNil)
		So(isValid, ShouldBeFalse)
		So(opFrame.GetResult().Result.MustTr().MustPaymentResult().Code, ShouldEqual, xdr.PaymentResultCodePaymentMalformed)
		So(opFrame.GetResult().Info.GetError(), ShouldEqual, errorData)
	})
	outLimitsValidator.On("VerifyLimits").Return(nil, nil)
	Convey("Failed to validate in limits", t, func() {
		errorData := "limits_failed"
		inLimitsValidator.On("VerifyLimits").Return(nil, errors.New(errorData)).Once()
		isValid, err := opFrame.CheckValid(historyQMock, coreQMock, &config)
		So(err.Error(), ShouldEqual, errorData)
		So(isValid, ShouldBeFalse)
	})
	Convey("Outlimits exceeded", t, func() {
		errorData := "account_restricted"
		inLimitsValidator.On("VerifyLimits").Return(&results.ExceededLimitError{
			Description: errorData,
		}, nil).Once()
		isValid, err := opFrame.CheckValid(historyQMock, coreQMock, &config)
		So(err, ShouldBeNil)
		So(isValid, ShouldBeFalse)
		So(opFrame.GetResult().Result.MustTr().MustPaymentResult().Code, ShouldEqual, xdr.PaymentResultCodePaymentMalformed)
		So(opFrame.GetResult().Info.GetError(), ShouldEqual, errorData)
	})
	inLimitsValidator.On("VerifyLimits").Return(nil, nil)
	Convey("Success", t, func() {
		isValid, err := opFrame.CheckValid(historyQMock, coreQMock, &config)
		So(err, ShouldBeNil)
		So(isValid, ShouldBeTrue)
		So(opFrame.GetResult().Result.MustTr().MustPaymentResult().Code, ShouldEqual, xdr.PaymentResultCodePaymentSuccess)
	})

}

func GetPaymentOpFrame(opFrame *OperationFrame) *PaymentOpFrame {
	innerOp, err := opFrame.GetInnerOp()
	if err != nil || innerOp == nil {
		log.Panic("Failed to create innerOp")
	}

	return innerOp.(*PaymentOpFrame)
}
