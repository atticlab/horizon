package transactions

import (
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/config"
	"bitbucket.org/atticlab/horizon/db2/core"
	"bitbucket.org/atticlab/horizon/db2/history"
)

type CreateAccountOpFrame struct {
	OperationFrame
	operation xdr.CreateAccountOp
}

func NewCreateAccountOpFrame(opFrame OperationFrame) *CreateAccountOpFrame {
	return &CreateAccountOpFrame{
		OperationFrame: opFrame,
		operation:      opFrame.Op.Body.MustCreateAccountOp(),
	}
}

func (frame *CreateAccountOpFrame) DoCheckValid(historyQ history.QInterface, coreQ core.QInterface, config *config.Config) (bool, error) {
	frame.getInnerResult().Code = xdr.CreateAccountResultCodeCreateAccountSuccess
	return true, nil
}

func (frame *CreateAccountOpFrame) getInnerResult() *xdr.CreateAccountResult {
	if frame.Result.Result.Tr.CreateAccountResult == nil {
		frame.Result.Result.Tr.CreateAccountResult = &xdr.CreateAccountResult{}
	}
	return frame.Result.Result.Tr.CreateAccountResult
}
