package transactions

import (
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/config"
	"bitbucket.org/atticlab/horizon/db2/core"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/txsub/results"
)

type CreatePassiveOfferOpFrame struct {
	OperationFrame
	createPassiveOffer xdr.CreatePassiveOfferOp
}

func NewCreatePassiveOfferOpFrame(opFrame OperationFrame) *CreatePassiveOfferOpFrame {
	return &CreatePassiveOfferOpFrame{
		OperationFrame:     opFrame,
		createPassiveOffer: opFrame.Op.Body.MustCreatePassiveOfferOp(),
	}
}

func (frame *CreatePassiveOfferOpFrame) DoCheckValid(historyQ history.QInterface, coreQ core.QInterface, config *config.Config) (bool, error) {
	manageOffer := frame.createManageOfferFrame()
	isValid, err := manageOffer.DoCheckValid(historyQ, coreQ, config)
	frame.Result.Info = manageOffer.Result.Info
	innerResult := frame.getInnerResult()
	manageOfferInnerResult := manageOffer.getInnerResult()
	*innerResult = *manageOfferInnerResult
	return isValid, err
}

func (frame *CreatePassiveOfferOpFrame) getInnerResult() *xdr.ManageOfferResult {
	if frame.Result.Result.Tr.CreatePassiveOfferResult == nil {
		frame.Result.Result.Tr.CreatePassiveOfferResult = &xdr.ManageOfferResult{}
	}
	return frame.Result.Result.Tr.CreatePassiveOfferResult
}

func (p *CreatePassiveOfferOpFrame) createManageOfferFrame() *ManageOfferOpFrame {
	resultOp := xdr.Operation{
		SourceAccount: p.Op.SourceAccount,
		Body: xdr.OperationBody{
			Type: xdr.OperationTypeManageOffer,
			ManageOfferOp: &xdr.ManageOfferOp{
				Selling: p.createPassiveOffer.Selling,
				Buying:  p.createPassiveOffer.Buying,
				Amount:  p.createPassiveOffer.Amount,
				Price:   p.createPassiveOffer.Price,
				OfferId: xdr.Uint64(0),
			},
		},
	}
	return NewManageOfferOpFrame(OperationFrame{
		Op:       &resultOp,
		ParentTx: p.ParentTx,
		Result: &results.OperationResult{
			Result: xdr.OperationResult{
				Code: xdr.OperationResultCodeOpInner,
				Tr: &xdr.OperationResultTr{
					Type: resultOp.Body.Type,
				},
			},
		},
	})
}
