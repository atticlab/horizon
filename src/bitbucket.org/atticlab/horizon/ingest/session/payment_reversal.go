package session

import (
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/log"
	"time"
)

func (is *Session) ingestPaymentReversal(storedPaymentID int64, reversalSourceAddress, paymentSourceAddress, assetCode string, amount xdr.Int64) error {
	logger := log.WithField("service", "payment_reversal_ingester")

	reversalSource, err := is.Ingestion.HistoryAccountCache.Get(reversalSourceAddress)
	if err != nil {
		logger.WithError(err).Error("Failed to get reversal source")
		return err
	}

	paymentSource, err := is.Ingestion.HistoryAccountCache.Get(paymentSourceAddress)
	if err != nil {
		logger.WithError(err).Error("Failed to get payment source for payment reversal")
		return err
	}

	var storedOp history.Operation
	err = is.Ingestion.HistoryQ().OperationByID(&storedOp, storedPaymentID)
	if err != nil {
		logger.WithError(err).Error("Failed to get stored payment for payment reversal")
		return err
	}

	now := time.Now()
	err = is.Ingestion.UpdateStatistics(reversalSource.Address, assetCode, paymentSource.AccountType, -int64(amount), storedOp.ClosedAt, now, true)
	if err != nil {
		return err
	}

	return is.Ingestion.UpdateStatistics(paymentSource.Address, assetCode, reversalSource.AccountType, -int64(amount), storedOp.ClosedAt, now, false)
}
