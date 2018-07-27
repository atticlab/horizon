package transactions

import (
    "github.com/atticlab/go-smart-base/xdr"
    "github.com/atticlab/horizon/db2/history"
    "github.com/atticlab/horizon/txsub/results"
    "github.com/atticlab/horizon/txsub/transactions/validators"
    "errors"
)

type ExternalPaymentOpFrame struct {
    *OperationFrame
    payment                   xdr.ExternalPaymentOp

    accountTypeValidator      validators.AccountTypeValidatorInterface
    assetsValidator           validators.AssetsValidatorInterface
    defaultOutLimitsValidator validators.OutgoingLimitsValidatorInterface
    defaultInLimitsValidator  validators.IncomingLimitsValidatorInterface
    traitsValidator           validators.TraitsValidatorInterface

    pathPayment               *PathPaymentOpFrame
}

func NewExternalPaymentOpFrame(opFrame *OperationFrame) *ExternalPaymentOpFrame {
    return &ExternalPaymentOpFrame{
        OperationFrame: opFrame,
        payment:        opFrame.Op.Body.MustExternalPaymentOp(),
    }
}

func (p *ExternalPaymentOpFrame) createPathPayment(manager *Manager) *PathPaymentOpFrame {
    op := xdr.Operation{
        SourceAccount: p.Op.SourceAccount,
        Body: xdr.OperationBody{
            Type: xdr.OperationTypePathPayment,
            PathPaymentOp: &xdr.PathPaymentOp{
                SendAsset:   p.payment.Asset,
                SendMax:     p.payment.Amount,
                Destination: p.payment.ExchangeAgent,
                DestAsset:   p.payment.Asset,
                DestAmount:  p.payment.Amount,
            },
        },
    }
    opFrame := NewOperationFrame(&op, p.ParentTxFrame, p.Index, *p.now)
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
    ppayment.assetsValidator = p.GetAssetsValidator(manager.HistoryQ)
    ppayment.traitsValidator = p.GetTraitsValidator()
    ppayment.defaultOutLimitsValidator = p.defaultOutLimitsValidator
    ppayment.defaultInLimitsValidator = p.defaultInLimitsValidator
    return ppayment
}

func (p *ExternalPaymentOpFrame) DoCheckValid(manager *Manager) (bool, error) {
    // Creating path payment op
    p.pathPayment = p.createPathPayment(manager)
    isValid, err := p.pathPayment.DoCheckValid(manager)
    if err != nil {
        return isValid, err
    }

    // Check if exchange agent exists
    if !p.pathPayment.isDestExists {
        p.getInnerResult().Code = xdr.PaymentResultCodePaymentNoDestination
        return false, nil
    }

    if !isValid {
        p.Result.Info = p.pathPayment.Result.Info

        var code xdr.PaymentResultCode
        switch p.pathPayment.getInnerResult().Code {
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

func (p *ExternalPaymentOpFrame) GetAccountTypeValidator() validators.AccountTypeValidatorInterface {
    if p.accountTypeValidator == nil {
        p.accountTypeValidator = validators.NewAccountTypeValidator()
    }
    return p.accountTypeValidator
}

func (p *ExternalPaymentOpFrame) getInnerResult() *xdr.PaymentResult {
    if p.Result.Result.Tr.PaymentResult == nil {
        p.Result.Result.Tr.PaymentResult = &xdr.PaymentResult{}
    }
    return p.Result.Result.Tr.PaymentResult
}

func (p *ExternalPaymentOpFrame) GetAssetsValidator(historyQ history.QInterface) validators.AssetsValidatorInterface {
    if p.assetsValidator == nil {
        p.log.Debug("Creating new assets validator")
        p.assetsValidator = validators.NewAssetsValidator(historyQ)
    }
    return p.assetsValidator
}

func (p *ExternalPaymentOpFrame) GetTraitsValidator() validators.TraitsValidatorInterface {
    if p.traitsValidator == nil {
        p.traitsValidator = validators.NewTraitsValidator()
    }
    return p.traitsValidator
}

func (p *ExternalPaymentOpFrame) DoRollbackCachedData(manager *Manager) error {
    return p.pathPayment.DoRollbackCachedData(manager)
}
