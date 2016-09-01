package txsub

import (
	"bitbucket.org/atticlab/horizon/log"
	"bitbucket.org/atticlab/horizon/render/problem"
	"bitbucket.org/atticlab/horizon/txsub/transactions"
)

type TransactionValidatorInterface interface {
	CheckTransaction(envelopeInfo *transactions.EnvelopeInfo) error
}

type TransactionValidator struct {
	manager *transactions.Manager
	log     *log.Entry
}

func NewTransactionValidator(manager *transactions.Manager) *TransactionValidator {
	return &TransactionValidator{
		manager: manager,
		log:     log.WithField("service", "transaction_validator"),
	}
}

// Validates transaction and operations
func (v *TransactionValidator) CheckTransaction(envelopeInfo *transactions.EnvelopeInfo) error {
	txFrame := transactions.NewTransactionFrame(envelopeInfo)
	isValid, err := txFrame.CheckValid(v.manager)
	if err != nil {
		v.log.WithStack(err).WithError(err).Error("Failed to validate tx")
		return &problem.ServerError
	}

	if !isValid {
		return txFrame.GetResult()
	}
	return nil
}
