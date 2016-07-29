package transactions

import (
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/db2/core"
	"bitbucket.org/atticlab/horizon/config"
)

type AccountMergeOpFrame struct {
	OperationFrame
	operation xdr.AccountId
}

func NewAccountMergeOpFrame(opFrame OperationFrame) *AccountMergeOpFrame {
	return &AccountMergeOpFrame{
		OperationFrame: opFrame,
		operation: opFrame.Op.Body.MustDestination(),
	}
}

func (frame *AccountMergeOpFrame) DoCheckValid(historyQ history.QInterface, coreQ core.QInterface, config *config.Config) (bool, error) {
	frame.getInnerResult().Code = xdr.AccountMergeResultCodeAccountMergeSuccess
	return true, nil
}

func (frame *AccountMergeOpFrame) getInnerResult() *xdr.AccountMergeResult {
	if frame.Result.Result.Tr.AccountMergeResult == nil {
		frame.Result.Result.Tr.AccountMergeResult = &xdr.AccountMergeResult{}
	}
	return frame.Result.Result.Tr.AccountMergeResult
}
