package transactions

import (
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/config"
	"bitbucket.org/atticlab/horizon/db2/core"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/log"
	"bitbucket.org/atticlab/horizon/txsub/results"
)

type TransactionFrame struct {
	tx       *xdr.TransactionEnvelope
	txResult *results.RestrictedTransactionError
	log      *log.Entry
}

func NewTransactionFrame(tx *xdr.TransactionEnvelope) *TransactionFrame {
	return &TransactionFrame{
		tx:  tx,
		log: log.WithField("service", "transaction_frame"),
	}
}

func (t *TransactionFrame) CheckValid(historyQ history.QInterface, coreQ core.QInterface, conf *config.Config) (bool, error) {
	t.log.Debug("Checking transaction")
	isTxValid, err := t.checkTransaction()
	if !isTxValid || err != nil {
		return isTxValid, err
	}

	t.log.Debug("Transaction is valid. Checking operations.")
	return t.checkOperations(historyQ, coreQ, conf)
}

func (t *TransactionFrame) checkTransaction() (bool, error) {
	// transaction can only have one adminOp
	if len(t.tx.Tx.Operations) == 1 {
		return true, nil
	}

	for _, op := range t.tx.Tx.Operations {
		if op.Body.Type == xdr.OperationTypeAdministrative {
			var err error
			t.txResult, err = results.NewRestrictedTransactionErrorTx(xdr.TransactionResultCodeTxFailed, results.AdditionalErrorInfoStrError("Administrative op must be only op in tx"))
			if err != nil {
				return false, err
			}
			return false, nil
		}
	}

	return true, nil
}

func (t *TransactionFrame) checkOperations(historyQ history.QInterface, coreQ core.QInterface, conf *config.Config) (bool, error) {
	opFrames := make([]OperationFrame, len(t.tx.Tx.Operations))
	isValid := true
	for i, op := range t.tx.Tx.Operations {
		opFrames[i] = NewOperationFrame(&op, t.tx)
		isOpValid, err := opFrames[i].CheckValid(historyQ, coreQ, conf)
		// failed to validate
		if err != nil {
			t.log.WithField("operation_i", i).Error("Failed to validate")
			return false, err
		}

		if !isOpValid {
			t.log.WithField("operation_i", i).WithField("result", opFrames[i].GetResult()).Debug("Is not valid")
			isValid = false
		}
	}
	if !isValid {
		var err error
		t.txResult, err = t.makeFailedTxResult(opFrames)
		if err != nil {
			t.log.Error("Failed to makeFailedTxResult")
			return false, err
		}
		return false, nil
	}
	return isValid, nil
}

func (t *TransactionFrame) makeFailedTxResult(opFrames []OperationFrame) (*results.RestrictedTransactionError, error) {
	operationResults := make([]results.OperationResult, len(opFrames))
	for i := range opFrames {
		operationResults[i] = opFrames[i].GetResult()
	}
	return results.NewRestrictedTransactionErrorOp(xdr.TransactionResultCodeTxFailed, operationResults)
}

// returns nil, if tx is successful
func (t *TransactionFrame) GetResult() *results.RestrictedTransactionError {
	return t.txResult
}
