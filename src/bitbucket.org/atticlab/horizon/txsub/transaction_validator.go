package txsub

import (
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/config"
	"bitbucket.org/atticlab/horizon/db2/core"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/log"
	"bitbucket.org/atticlab/horizon/render/problem"
	"bitbucket.org/atticlab/horizon/txsub/transactions"
)

type TransactionValidatorInterface interface {
	CheckTransaction(tx *xdr.TransactionEnvelope) error
}

type TransactionValidator struct {
	historyQ history.QInterface
	coreQ    core.QInterface
	config   *config.Config
	log      *log.Entry
}

func NewTransactionValidator(historyQ history.QInterface, coreQ core.QInterface, config *config.Config) *TransactionValidator {
	return &TransactionValidator{
		historyQ: historyQ,
		coreQ:    coreQ,
		config:   config,
		log:      log.WithField("service", "transaction_validator"),
	}
}

// Validates transaction and operations
func (v *TransactionValidator) CheckTransaction(tx *xdr.TransactionEnvelope) error {
	txFrame := transactions.NewTransactionFrame(tx)
	isValid, err := txFrame.CheckValid(v.historyQ, v.coreQ, v.config)
	if err != nil {
		v.log.WithStack(err).WithError(err).Error("Failed to validate tx")
		return &problem.ServerError
	}

	if !isValid {
		return txFrame.GetResult()
	}
	return nil
}
