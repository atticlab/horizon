package transactions

import (
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/txsub/results"
	"bitbucket.org/atticlab/horizon/txsub/transactions/validators"
)

type ManageOfferOpFrame struct {
	OperationFrame
	manageOffer xdr.ManageOfferOp
}

func NewManageOfferOpFrame(opFrame OperationFrame) *ManageOfferOpFrame {
	return &ManageOfferOpFrame{
		OperationFrame: opFrame,
		manageOffer:    opFrame.Op.Body.MustManageOfferOp(),
	}
}

func (frame *ManageOfferOpFrame) DoCheckValid(manager *Manager) (bool, error) {
	isValid, err := validators.NewAssetsValidator(manager.HistoryQ).IsAssetsValid(frame.manageOffer.Buying, frame.manageOffer.Selling)
	if err != nil {
		return false, err
	}

	if !isValid {
		frame.getInnerResult().Code = xdr.ManageOfferResultCodeManageOfferMalformed
		frame.Result.Info = results.AdditionalErrorInfoError(ASSET_NOT_ALLOWED)
		return false, nil
	}
	frame.getInnerResult().Code = xdr.ManageOfferResultCodeManageOfferSuccess
	return true, nil
}

func (frame *ManageOfferOpFrame) getInnerResult() *xdr.ManageOfferResult {
	if frame.Result.Result.Tr.ManageOfferResult == nil {
		frame.Result.Result.Tr.ManageOfferResult = &xdr.ManageOfferResult{}
	}
	return frame.Result.Result.Tr.ManageOfferResult
}
