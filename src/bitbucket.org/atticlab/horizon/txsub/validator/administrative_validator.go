package validator

import (
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/admin"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/log"
	"bitbucket.org/atticlab/horizon/render/problem"
	"bitbucket.org/atticlab/horizon/txsub/results"
	"encoding/json"
)

var (
	malformed_admin_op = results.NewAdminOpResult(xdr.AdministrativeResultCodeAdministrativeMalformed)
)

type AdministrativeValidator struct {
	log       *log.Entry
	historyDb *history.Q
}

func NewAdministrativeValidator(historyDb *history.Q) *AdministrativeValidator {
	return &AdministrativeValidator{
		log:       log.WithField("service", "limits_validator"),
		historyDb: historyDb,
	}
}

func (v *AdministrativeValidator) CheckTransaction(tx *xdr.TransactionEnvelope) (*results.RestrictedTransactionError, error) {
	if len(tx.Tx.Operations) == 1 {
		return nil, nil
	}
	for _, op := range tx.Tx.Operations {
		if op.Body.Type == xdr.OperationTypeAdministrative {
			return results.NewRestrictedTransactionErrorTx(xdr.TransactionResultCodeTxFailed, results.AdditionalErrorInfoStrError("Administrative op must be only op in tx"))
		}
	}
	return nil, nil
}

// checkAccountTypes Parse tx and check account types
func (v *AdministrativeValidator) CheckOperation(source string, op *xdr.Operation) (opResult xdr.OperationResult, additionalInfo results.AdditionalErrorInfo, err error) {
	if op.Body.Type != xdr.OperationTypeAdministrative {
		opResult, err := results.GetSuccessResult(op.Body.Type)
		if err != nil {
			return opResult, nil, err
		}

		return opResult, nil, nil
	}
	adminOp := op.Body.MustAdminOp()
	var opData map[string]interface{}
	err = json.Unmarshal([]byte(adminOp.OpData), &opData)
	if err != nil {
		return malformed_admin_op, results.AdditionalErrorInfoStrError(err.Error()), nil
	}

	adminActionProvider := admin.NewAdminActionProvider(v.historyDb)
	adminAction, err := adminActionProvider.CreateNewParser(opData)
	if err != nil {
		return malformed_admin_op, results.AdditionalErrorInfoStrError(err.Error()), nil
	}

	adminAction.Validate()
	err = adminAction.GetError()
	if err != nil {
		switch err.(type) {
		case *admin.InvalidFieldError:
			invalidField := err.(*admin.InvalidFieldError)
			return malformed_admin_op, results.AdditionalErrorInfoError(invalidField), nil
		case *problem.P:
			prob := err.(*problem.P)
			if prob.Type == problem.ServerError.Type {
				return opResult, nil, err
			}
			return malformed_admin_op, results.AdditionalErrorInfoError(err), nil
		default:
			return malformed_admin_op, results.AdditionalErrorInfoError(err), nil
		}
	}
	return results.NewAdminOpResult(xdr.AdministrativeResultCodeAdministrativeSuccess), nil, nil
}
