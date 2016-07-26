package transactions

import (
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/txsub/results"
	"bitbucket.org/atticlab/horizon/txsub/transactions/validators"
	"bitbucket.org/atticlab/horizon/db2/core"
	"bitbucket.org/atticlab/horizon/config"
)

type ChangeTrustOpFrame struct {
	OperationFrame
	operation xdr.ChangeTrustOp
}

func NewChangeTrustOpFrame(opFrame OperationFrame) *ChangeTrustOpFrame {
	return &ChangeTrustOpFrame{
		OperationFrame: opFrame,
		operation:      opFrame.Op.Body.MustChangeTrustOp(),
	}
}

func (frame *ChangeTrustOpFrame) DoCheckValid(historyQ history.QInterface, coreQ core.QInterface, config *config.Config) (bool, error) {
	isValid, err := validators.NewAssetsValidator(historyQ).IsAssetValid(frame.operation.Line)
	if err != nil {
		return false, err
	}

	if !isValid {
		frame.getInnerResult().Code = xdr.ChangeTrustResultCodeChangeTrustMalformed
		frame.Result.Info = results.AdditionalErrorInfoError(ASSET_NOT_ALLOWED)
		return false, nil
	}
	frame.getInnerResult().Code = xdr.ChangeTrustResultCodeChangeTrustSuccess
	return true, nil
}

func (frame *ChangeTrustOpFrame) getInnerResult() *xdr.ChangeTrustResult {
	if frame.Result.Result.Tr.ChangeTrustResult == nil {
		frame.Result.Result.Tr.ChangeTrustResult = &xdr.ChangeTrustResult{}
	}
	return frame.Result.Result.Tr.ChangeTrustResult
}
