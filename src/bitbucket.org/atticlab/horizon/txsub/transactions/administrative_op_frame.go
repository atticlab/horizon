package transactions

import (
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/admin"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/render/problem"
	"bitbucket.org/atticlab/horizon/txsub/results"
	"encoding/json"
	"bitbucket.org/atticlab/horizon/db2/core"
	"bitbucket.org/atticlab/horizon/config"
)

type AdministrativeOpFrame struct {
	OperationFrame
	operation xdr.AdministrativeOp
	adminActionProvider admin.AdminActionProviderInterface
}

func NewAdministrativeOpFrame(opFrame OperationFrame) *AdministrativeOpFrame {
	return &AdministrativeOpFrame{
		OperationFrame: opFrame,
		operation:      opFrame.Op.Body.MustAdminOp(),
	}
}

func (frame *AdministrativeOpFrame) getAdminActionProvider(historyQ history.QInterface) admin.AdminActionProviderInterface {
	if frame.adminActionProvider == nil {
		frame.adminActionProvider = admin.NewAdminActionProvider(historyQ)
	}
	return frame.adminActionProvider
}

func (frame *AdministrativeOpFrame) DoCheckValid(historyQ history.QInterface, coreQ core.QInterface, config *config.Config) (bool, error) {
	var opData map[string]interface{}
	err := json.Unmarshal([]byte(frame.operation.OpData), &opData)
	if err != nil {
		frame.getInnerResult().Code = xdr.AdministrativeResultCodeAdministrativeMalformed
		frame.Result.Info = results.AdditionalErrorInfoError(err)
		return false, nil
	}

	adminAction, err := frame.getAdminActionProvider(historyQ).CreateNewParser(opData)
	if err != nil {
		frame.getInnerResult().Code = xdr.AdministrativeResultCodeAdministrativeMalformed
		frame.Result.Info = results.AdditionalErrorInfoError(err)
		return false, nil
	}

	adminAction.Validate()
	err = adminAction.GetError()
	if err != nil {
		switch err.(type) {
		case *admin.InvalidFieldError:
			frame.getInnerResult().Code = xdr.AdministrativeResultCodeAdministrativeMalformed
			invalidField := err.(*admin.InvalidFieldError)
			frame.Result.Info = results.AdditionalErrorInfoInvField(*invalidField)
			return false, nil
		case *problem.P:
			prob := err.(*problem.P)
			if prob.Type == problem.ServerError.Type {
				return false, err
			}
			frame.getInnerResult().Code = xdr.AdministrativeResultCodeAdministrativeMalformed
			frame.Result.Info = results.AdditionalErrorInfoError(err)
			return false, nil
		default:
			return false, err
		}
	}
	frame.getInnerResult().Code = xdr.AdministrativeResultCodeAdministrativeSuccess
	return true, nil
}

func (frame *AdministrativeOpFrame) getInnerResult() *xdr.AdministrativeResult {
	if frame.Result.Result.Tr.AdminResult == nil {
		frame.Result.Result.Tr.AdminResult = &xdr.AdministrativeResult{}
	}
	return frame.Result.Result.Tr.AdminResult
}
