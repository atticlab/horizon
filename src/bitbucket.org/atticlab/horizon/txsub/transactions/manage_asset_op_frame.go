package transactions

import (
	"bitbucket.org/atticlab/go-smart-base/xdr"
)

type ManageAssetOpFrame struct {
	*OperationFrame
	operation xdr.ManageAssetOp
}

func NewManageAssetOpFrame(opFrame *OperationFrame) *ManageAssetOpFrame {
	return &ManageAssetOpFrame{
		OperationFrame: opFrame,
		operation:      opFrame.Op.Body.MustManageAssetOp(),
	}
}

func (frame *ManageAssetOpFrame) DoCheckValid(manager *Manager) (bool, error) {
	frame.getInnerResult().Code = xdr.ManageAssetResultCodeManageAssetSuccess
	return true, nil
}

func (frame *ManageAssetOpFrame) getInnerResult() *xdr.ManageAssetResult {
	if frame.Result.Result.Tr.ManageAssetResult == nil {
		frame.Result.Result.Tr.ManageAssetResult = &xdr.ManageAssetResult{}
	}
	return frame.Result.Result.Tr.ManageAssetResult
}
