package transactions

import (
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/config"
	"bitbucket.org/atticlab/horizon/db2/core"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/txsub/results"
	"bitbucket.org/atticlab/horizon/txsub/transactions/validators"
	"errors"
)

type PaymentOpFrame struct {
	OperationFrame
	payment xdr.PaymentOp

	accountTypeValidator      validators.AccountTypeValidatorInterface
	assetsValidator           validators.AssetsValidatorInterface
	defaultOutLimitsValidator validators.OutgoingLimitsValidatorInterface
	defaultInLimitsValidator  validators.IncomingLimitsValidatorInterface
	traitsValidator           validators.TraitsValidatorInterface
}

func (p *PaymentOpFrame) GetAccountTypeValidator() validators.AccountTypeValidatorInterface {
	if p.accountTypeValidator == nil {
		p.accountTypeValidator = validators.NewAccountTypeValidator()
	}
	return p.accountTypeValidator
}

func NewPaymentOpFrame(opFrame OperationFrame) *PaymentOpFrame {
	return &PaymentOpFrame{
		OperationFrame: opFrame,
		payment:        opFrame.Op.Body.MustPaymentOp(),
	}
}

func (p *PaymentOpFrame) DoCheckValid(historyQ history.QInterface, coreQ core.QInterface, config *config.Config) (bool, error) {
	//creating path payment op
	ppayment := p.createPathPayment(historyQ)
	isValid, err := ppayment.DoCheckValid(historyQ, coreQ, config)
	if err != nil {
		return isValid, err
	}

	if !isValid {
		p.Result.Info = ppayment.Result.Info

		var code xdr.PaymentResultCode
		switch ppayment.getInnerResult().Code {
		case xdr.PathPaymentResultCodePathPaymentMalformed:
			code = xdr.PaymentResultCodePaymentMalformed
		case xdr.PathPaymentResultCodePathPaymentUnderfunded:
			code = xdr.PaymentResultCodePaymentUnderfunded
		case xdr.PathPaymentResultCodePathPaymentSrcNoTrust:
			code = xdr.PaymentResultCodePaymentSrcNoTrust
		case xdr.PathPaymentResultCodePathPaymentSrcNotAuthorized:
			code = xdr.PaymentResultCodePaymentSrcNotAuthorized
		case xdr.PathPaymentResultCodePathPaymentNoDestination:
			code = xdr.PaymentResultCodePaymentNoDestination
		case xdr.PathPaymentResultCodePathPaymentNoTrust:
			code = xdr.PaymentResultCodePaymentNoTrust
		case xdr.PathPaymentResultCodePathPaymentNotAuthorized:
			code = xdr.PaymentResultCodePaymentNotAuthorized
		case xdr.PathPaymentResultCodePathPaymentLineFull:
			code = xdr.PaymentResultCodePaymentLineFull
		case xdr.PathPaymentResultCodePathPaymentNoIssuer:
			code = xdr.PaymentResultCodePaymentNoIssuer
		default:
			return false, errors.New("Unexpected error code from pathPayment")
		}
		p.getInnerResult().Code = code
		return false, nil
	}

	p.getInnerResult().Code = xdr.PaymentResultCodePaymentSuccess
	return true, nil
}

func (p *PaymentOpFrame) getInnerResult() *xdr.PaymentResult {
	if p.Result.Result.Tr.PaymentResult == nil {
		p.Result.Result.Tr.PaymentResult = &xdr.PaymentResult{}
	}
	return p.Result.Result.Tr.PaymentResult
}

func (p *PaymentOpFrame) createPathPayment(historyQ history.QInterface) *PathPaymentOpFrame {
	p.log.WithField("createPathPayment", p.SourceAccount.Accountid).Debug("ASD")
	op := xdr.Operation{
		SourceAccount: p.Op.SourceAccount,
		Body: xdr.OperationBody{
			Type: xdr.OperationTypePathPayment,
			PathPaymentOp: &xdr.PathPaymentOp{
				SendAsset:   p.payment.Asset,
				SendMax:     p.payment.Amount,
				Destination: p.payment.Destination,
				DestAsset:   p.payment.Asset,
				DestAmount:  p.payment.Amount,
			},
		},
	}
	opFrame := NewOperationFrame(&op, p.ParentTx)
	opFrame.Result = &results.OperationResult{
		Result: xdr.OperationResult{
			Code: xdr.OperationResultCodeOpInner,
			Tr: &xdr.OperationResultTr{
				Type: opFrame.Op.Body.Type,
			},
		},
	}
	opFrame.SourceAccount = p.SourceAccount
	opFrame.innerOp = nil
	innerOp, _ := opFrame.GetInnerOp()
	ppayment := innerOp.(*PathPaymentOpFrame)
	ppayment.accountTypeValidator = p.GetAccountTypeValidator()
	ppayment.assetsValidator = p.GetAssetsValidator(historyQ)
	ppayment.traitsValidator = p.GetTraitsValidator(historyQ)
	ppayment.defaultOutLimitsValidator = p.defaultOutLimitsValidator
	ppayment.defaultInLimitsValidator = p.defaultInLimitsValidator
	p.log.WithField("createPathPayment", ppayment.SourceAccount.Accountid).Debug("ASD")
	return ppayment
}

func (p *PaymentOpFrame) GetAssetsValidator(historyQ history.QInterface) validators.AssetsValidatorInterface {
	if p.assetsValidator == nil {
		p.log.Debug("Creating new assets validator")
		p.assetsValidator = validators.NewAssetsValidator(historyQ)
	}
	return p.assetsValidator
}

func (p *PaymentOpFrame) GetTraitsValidator(historyQ history.QInterface) validators.TraitsValidatorInterface {
	if p.traitsValidator == nil {
		p.traitsValidator = validators.NewTraitsValidator(historyQ)
	}
	return p.traitsValidator
}
