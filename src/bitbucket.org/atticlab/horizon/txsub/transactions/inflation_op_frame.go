package transactions

import (
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/config"
	"bitbucket.org/atticlab/horizon/db2/core"
)

type InflationOpFrame struct {
	OperationFrame
}

func NewInflationOpFrame(opFrame OperationFrame) *InflationOpFrame {
	return &InflationOpFrame{
		OperationFrame: opFrame,
	}
}

func (frame *InflationOpFrame) DoCheckValid(historyQ history.QInterface, coreQ core.QInterface, config *config.Config) (bool, error) {
	frame.getInnerResult().Code = xdr.InflationResultCodeInflationSuccess
	return true, nil
}

func (frame *InflationOpFrame) getInnerResult() *xdr.InflationResult {
	if frame.Result.Result.Tr.InflationResult == nil {
		frame.Result.Result.Tr.InflationResult = &xdr.InflationResult{}
	}
	return frame.Result.Result.Tr.InflationResult
}
