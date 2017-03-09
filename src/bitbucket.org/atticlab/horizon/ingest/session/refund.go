package session

import (
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/log"
	"time"
)

func (is *Session) ingestRefund(storedPaymentID int64, refundSourceAddress, paymentSourceAddress, assetCode string, amount xdr.Int64, originalAmount xdr.Int64) error {
	logger := log.WithField("service", "payment_refund_ingester")

	refundSource, err := is.Ingestion.HistoryAccountCache.Get(refundSourceAddress)
	if err != nil {
		logger.WithError(err).Error("Failed to get refund source")
		return err
	}

	paymentSource, err := is.Ingestion.HistoryAccountCache.Get(paymentSourceAddress)
	if err != nil {
		logger.WithError(err).Error("Failed to get payment source for refund")
		return err
	}

	var storedOp history.Operation
	err = is.Ingestion.HistoryQ().OperationByID(&storedOp, storedPaymentID)
	if err != nil {
		logger.WithError(err).Error("Failed to get stored payment for refund")
		return err
	}

	now := time.Now()
	err = is.Ingestion.UpdateStatistics(refundSource.Address, assetCode, paymentSource.AccountType, -int64(amount), storedOp.ClosedAt, now, true)
	if err != nil {
		return err
	}

	return is.Ingestion.UpdateStatistics(paymentSource.Address, assetCode, refundSource.AccountType, -int64(amount), storedOp.ClosedAt, now, false)
}
