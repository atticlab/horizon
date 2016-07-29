package transactions

import (
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/db2/core"
	"bitbucket.org/atticlab/horizon/config"
)

type ManageDataOpFrame struct {
	OperationFrame
	operation xdr.ManageDataOp
}

func NewManageDataOpFrame(opFrame OperationFrame) *ManageDataOpFrame {
	return &ManageDataOpFrame{
		OperationFrame: opFrame,
		operation:      opFrame.Op.Body.MustManageDataOp(),
	}
}

func (frame *ManageDataOpFrame) DoCheckValid(historyQ history.QInterface, coreQ core.QInterface, config *config.Config) (bool, error) {
	frame.getInnerResult().Code = xdr.ManageDataResultCodeManageDataSuccess
	return true, nil
}

func (frame *ManageDataOpFrame) getInnerResult() *xdr.ManageDataResult {
	if frame.Result.Result.Tr.ManageDataResult == nil {
		frame.Result.Result.Tr.ManageDataResult = &xdr.ManageDataResult{}
	}
	return frame.Result.Result.Tr.ManageDataResult
}
