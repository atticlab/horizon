package transactions

import (
	"bitbucket.org/atticlab/go-smart-base/amount"
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/db2/history/details"
	"database/sql"
	"time"
)

var MAX_REVERSE_TIME = time.Duration(24) * time.Hour

type PaymentReversalOpFrame struct {
	*OperationFrame
	paymentReversal xdr.PaymentReversalOp
}

func NewPaymentReversalOpFrame(opFrame *OperationFrame) *PaymentReversalOpFrame {
	return &PaymentReversalOpFrame{
		OperationFrame:  opFrame,
		paymentReversal: opFrame.Op.Body.MustPaymentReversalOp(),
	}
}

func (p *PaymentReversalOpFrame) DoCheckValid(manager *Manager) (bool, error) {
	if !p.isPaymentValid() {
		p.log.WithField("amount", int64(p.paymentReversal.Amount)).WithField("commission", int64(p.paymentReversal.CommissionAmount)).Debug("Reversal payment amount or commission amount is invalid")
		p.getInnerResult().Code = xdr.PaymentReversalResultCodePaymentReversalMalformed
		return false, nil
	}

	isValid, err := p.validateAgainstPayment(manager)
	if err != nil {
		p.log.WithError(err).Error("Failed to validate reversal against payment!")
		return false, err
	}

	if isValid {
		p.getInnerResult().Code = xdr.PaymentReversalResultCodePaymentReversalSuccess
	}

	return isValid, nil
}

func (p *PaymentReversalOpFrame) isPaymentValid() bool {
	return int64(p.paymentReversal.Amount) > 0 && int64(p.paymentReversal.CommissionAmount) >= 0
}

func (p *PaymentReversalOpFrame) validateAgainstPayment(manager *Manager) (bool, error) {
	var operation history.Operation
	err := manager.HistoryQ.OperationByID(&operation, int64(p.paymentReversal.PaymentId))
	if err != nil {
		if err != sql.ErrNoRows {
			p.log.WithError(err).Error("Failed to get payment from db")
			return false, err
		}
		// does not exists
		p.getInnerResult().Code = xdr.PaymentReversalResultCodePaymentReversalPaymentDoesNotExists
		return false, nil
	}

	if operation.Type != xdr.OperationTypePayment {
		p.getInnerResult().Code = xdr.PaymentReversalResultCodePaymentReversalPaymentDoesNotExists
		return false, nil
	}

	if operation.ClosedAt.Add(MAX_REVERSE_TIME).Before(*p.now) {
		p.getInnerResult().Code = xdr.PaymentReversalResultCodePaymentReversalPaymentExpired
		return false, nil
	}

	if operation.SourceAccount != p.paymentReversal.PaymentSource.Address() {
		p.getInnerResult().Code = xdr.PaymentReversalResultCodePaymentReversalInvalidSource
		return false, nil
	}

	var paymentDetails details.Payment
	err = operation.UnmarshalDetails(&paymentDetails)
	if err != nil {
		p.log.WithError(err).Error("Failed to get payment details!")
		return false, err
	}

	return p.validateReversalPaymentDetails(&paymentDetails), nil
}

func (p *PaymentReversalOpFrame) validateReversalPaymentDetails(paymentDetails *details.Payment) bool {
	if paymentDetails.To != p.SourceAccount.Address {
		p.getInnerResult().Code = xdr.PaymentReversalResultCodePaymentReversalInvalidPaymentSender
		return false
	}

	if int64(p.paymentReversal.Amount) != int64(amount.MustParse(paymentDetails.Amount)) {
		p.getInnerResult().Code = xdr.PaymentReversalResultCodePaymentReversalInvalidAmount
		return false
	}

	if !p.isCommissionValid(&paymentDetails.Fee) {
		p.getInnerResult().Code = xdr.PaymentReversalResultCodePaymentReversalInvalidCommission
		return false
	}

	if !p.isAssetValid(&paymentDetails.Asset) {
		p.getInnerResult().Code = xdr.PaymentReversalResultCodePaymentReversalInvalidAsset
		return false
	}

	return true
}

func (p *PaymentReversalOpFrame) isCommissionValid(fee *details.Fee) bool {
	commission := int64(p.paymentReversal.CommissionAmount)
	if fee.AmountCharged == nil {
		return commission == 0
	}

	actualCommission := int64(amount.MustParse(*fee.AmountCharged))
	return actualCommission == commission
}

func (p *PaymentReversalOpFrame) isAssetValid(asset *details.Asset) bool {
	if p.paymentReversal.Asset.Type == xdr.AssetTypeAssetTypeNative {
		return false
	}

	var t, code, i string
	err := p.paymentReversal.Asset.Extract(&t, &code, &i)
	if err != nil {
		return false
	}

	return asset.Type == t && asset.Code == code && asset.Issuer == i
}

func (p *PaymentReversalOpFrame) getInnerResult() *xdr.PaymentReversalResult {
	if p.Result.Result.Tr.PaymentReversalResult == nil {
		p.Result.Result.Tr.PaymentReversalResult = &xdr.PaymentReversalResult{}
	}
	return p.Result.Result.Tr.PaymentReversalResult
}

func (p *PaymentReversalOpFrame) DoRollbackCachedData(manager *Manager) error {
	return nil
}
