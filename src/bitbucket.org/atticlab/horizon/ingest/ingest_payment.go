package ingest

import (
	"bitbucket.org/atticlab/go-smart-base/amount"
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/log"
	"time"
)

func (is *Session) ingestPayment(from, to string, sourceAmount, destAmount xdr.Int64, sourceAsset, destAsset string) {
	var sourceType, destinationType xdr.AccountType

	is.Err = is.getAccountType(&sourceType, from)
	if is.Err != nil {
		return
	}

	is.Err = is.getAccountType(&destinationType, to)
	if is.Err != nil {
		return
	}

	if destinationType == xdr.AccountTypeAccountAnonymousUser {
		is.Err = is.Ingestion.Account(is.Cursor.OperationID(), to)
		if is.Err != nil {
			log.Error("Failed to ingest anonymous account created by payment!")
			return
		}
	}

	ledgerCloseTime := time.Unix(is.Cursor.Ledger().CloseTime, 0).Local()
	log.WithFields(log.F{
		"from":              from,
		"to":                to,
		"dest_type":         destinationType,
		"source_amount":     amount.String(sourceAmount),
		"dest_amount":       amount.String(destAmount),
		"source_asset":      sourceAsset,
		"dest_asset":        destAsset,
		"ledger_close_time": ledgerCloseTime,
	}).Info("Payment")

	now := time.Now()
	is.Err = is.Ingestion.UpdateAccountOutcome(from, sourceAsset, destinationType, int64(sourceAmount), ledgerCloseTime, now)
	if is.Err != nil {
		return
	}

	is.Err = is.Ingestion.UpdateAccountIncome(to, destAsset, sourceType, int64(destAmount), ledgerCloseTime, now)
}
