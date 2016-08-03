package txsub

import (
	"bitbucket.org/atticlab/horizon/config"
	"bitbucket.org/atticlab/horizon/db2/core"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/log"
	"bitbucket.org/atticlab/horizon/render/problem"
	"bitbucket.org/atticlab/horizon/txsub/transactions"
	"bitbucket.org/atticlab/horizon/txsub/transactions/statistics"
)

type TransactionValidatorInterface interface {
	CheckTransaction(envelopeInfo *transactions.EnvelopeInfo) error
}

type TransactionValidator struct {
	coreQ        core.QInterface
	historyQ     history.QInterface
	statsManager statistics.ManagerInterface
	config       *config.Config
	log          *log.Entry
}

func NewTransactionValidator(historyQ history.QInterface, coreQ core.QInterface, statsManager statistics.ManagerInterface, config *config.Config) *TransactionValidator {
	return &TransactionValidator{
		historyQ:     historyQ,
		coreQ:        coreQ,
		config:       config,
		log:          log.WithField("service", "transaction_validator"),
		statsManager: statsManager,
	}
}

// Validates transaction and operations
func (v *TransactionValidator) CheckTransaction(envelopeInfo *transactions.EnvelopeInfo) error {
	manager := transactions.NewManager(v.coreQ, v.historyQ, v.statsManager, v.config)
	txFrame := transactions.NewTransactionFrame(envelopeInfo)
	isValid, err := txFrame.CheckValid(manager)
	if err != nil {
		v.log.WithStack(err).WithError(err).Error("Failed to validate tx")
		return &problem.ServerError
	}

	if !isValid {
		return txFrame.GetResult()
	}
	return nil
}
