package transactions

import (
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/db2/core"
	"bitbucket.org/atticlab/horizon/config"
)

type SetOptionsOpFrame struct {
	OperationFrame
	operation xdr.SetOptionsOp
}

func NewSetOptionsOpFrame(opFrame OperationFrame) *SetOptionsOpFrame {
	return &SetOptionsOpFrame{
		OperationFrame: opFrame,
		operation:      opFrame.Op.Body.MustSetOptionsOp(),
	}
}

func (p *SetOptionsOpFrame) DoCheckValid(historyQ history.QInterface, coreQ core.QInterface, config *config.Config) (bool, error) {
	p.getInnerResult().Code = xdr.SetOptionsResultCodeSetOptionsSuccess
	return true, nil
}

func (p *SetOptionsOpFrame) getInnerResult() *xdr.SetOptionsResult {
	if p.Result.Result.Tr.SetOptionsResult == nil {
		p.Result.Result.Tr.SetOptionsResult = &xdr.SetOptionsResult{}
	}
	return p.Result.Result.Tr.SetOptionsResult
}
