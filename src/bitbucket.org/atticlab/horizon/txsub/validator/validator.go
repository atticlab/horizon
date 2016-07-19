package validator

import (
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/txsub/results"
)

type ValidatorInterface interface {
	// validates transaction. If tx is invalid returns results.RestrictedTransactionError
	CheckTransaction(tx *xdr.TransactionEnvelope) (*results.RestrictedTransactionError, error)
	// validates operation
	CheckOperation(sourceAccount string, op *xdr.Operation) (result xdr.OperationResult, additionalInfo results.AdditionalErrorInfo, err error)
}
