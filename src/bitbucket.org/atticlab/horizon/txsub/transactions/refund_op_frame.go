package transactions

import (
	"bitbucket.org/atticlab/go-smart-base/amount"
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/db2/history/details"
	"database/sql"
)


type RefundOpFrame struct {
	*OperationFrame
	refund xdr.RefundOp
}

func NewRefundOpFrame(opFrame *OperationFrame) *RefundOpFrame {
	return &RefundOpFrame{
		OperationFrame:  opFrame,
		refund: opFrame.Op.Body.MustRefundOp(),
	}
}

func (p *RefundOpFrame) DoCheckValid(manager *Manager) (bool, error) {
	if p.SourceAccount.AccountType != xdr.AccountTypeAccountMerchant {
		p.getInnerResult().Code = xdr.RefundResultCodeRefundNotAllowed
		return false, nil
	}
	if !p.isPaymentValid() {
		p.log.WithField("amount", int64(p.refund.Amount)).WithField("original_amount", int64(p.refund.OriginalAmount)).Debug("Refund amount or Original amount is invalid")
		p.getInnerResult().Code = xdr.RefundResultCodeRefundMalformed
		return false, nil
	}

	isValid, err := p.validateAgainstPayment(manager)
	if err != nil {
		p.log.WithError(err).Error("Failed to validate refund against payment!")
		return false, err
	}

	if isValid {
		p.getInnerResult().Code = xdr.RefundResultCodeRefundSuccess
	}

	return isValid, nil
}

func (p *RefundOpFrame) isPaymentValid() bool {
	return int64(p.refund.Amount) > 0 && int64(p.refund.OriginalAmount) > 0 && int64(p.refund.Amount) <= int64(p.refund.OriginalAmount)
}

func (p *RefundOpFrame) validateAgainstPayment(manager *Manager) (bool, error) {
	var operation history.Operation
	err := manager.HistoryQ.OperationByID(&operation, int64(p.refund.PaymentId))
	if err != nil {
		if err != sql.ErrNoRows {
			p.log.WithError(err).Error("Failed to get payment from db")
			return false, err
		}
		// does not exists
		p.getInnerResult().Code = xdr.RefundResultCodeRefundPaymentDoesNotExists
		return false, nil
	}

	if operation.Type != xdr.OperationTypePayment {
		p.getInnerResult().Code = xdr.RefundResultCodeRefundPaymentDoesNotExists
		return false, nil
	}

	if operation.SourceAccount != p.refund.PaymentSource.Address() {
		p.getInnerResult().Code = xdr.RefundResultCodeRefundInvalidSource
		return false, nil
	}

	var paymentDetails details.Payment
	err = operation.UnmarshalDetails(&paymentDetails)
	if err != nil {
		p.log.WithError(err).Error("Failed to get payment details!")
		return false, err
	}

	return p.validateRefundDetails(&paymentDetails), nil
}

func (p *RefundOpFrame) validateRefundDetails(paymentDetails *details.Payment) bool {
	if paymentDetails.To != p.SourceAccount.Address {
		p.getInnerResult().Code = xdr.RefundResultCodeRefundInvalidPaymentSender
		return false
	}

	if int64(p.refund.OriginalAmount) != int64(amount.MustParse(paymentDetails.Amount)) {
		p.getInnerResult().Code = xdr.RefundResultCodeRefundInvalidAmount
		return false
	}

	if !p.isAssetValid(&paymentDetails.Asset) {
		p.getInnerResult().Code = xdr.RefundResultCodeRefundInvalidAsset
		return false
	}

	return true
}


func (p *RefundOpFrame) isAssetValid(asset *details.Asset) bool {
	if p.refund.Asset.Type == xdr.AssetTypeAssetTypeNative {
		return false
	}

	var t, code, i string
	err := p.refund.Asset.Extract(&t, &code, &i)
	if err != nil {
		return false
	}

	return asset.Type == t && asset.Code == code && asset.Issuer == i
}

func (p *RefundOpFrame) getInnerResult() *xdr.RefundResult {
	if p.Result.Result.Tr.RefundResult == nil {
		p.Result.Result.Tr.RefundResult = &xdr.RefundResult{}
	}
	return p.Result.Result.Tr.RefundResult
}

func (p *RefundOpFrame) DoRollbackCachedData(manager *Manager) error {
	return nil
}
