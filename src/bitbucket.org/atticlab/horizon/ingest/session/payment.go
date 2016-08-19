package session

import (
	"bitbucket.org/atticlab/go-smart-base/amount"
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/log"
	"time"
)

func (is *Session) ingestPayment(from, to string, sourceAmount, destAmount xdr.Int64, sourceAsset, destAsset string) error {
	var sourceType, destinationType xdr.AccountType

	err := is.getAccountType(&sourceType, from)
	if err != nil {
		return err
	}

	err = is.getAccountType(&destinationType, to)
	if err != nil {
		return err
	}

	if destinationType == xdr.AccountTypeAccountAnonymousUser {
		err = is.Ingestion.Account(is.Cursor.OperationID(), to)
		if err != nil {
			log.Error("Failed to ingest anonymous account created by payment!")
			return err
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
	}).Debug("Payment")

	now := time.Now()
	err = is.Ingestion.UpdateAccountOutcome(from, sourceAsset, destinationType, int64(sourceAmount), ledgerCloseTime, now)
	if err != nil {
		return err
	}

	return is.Ingestion.UpdateAccountIncome(to, destAsset, sourceType, int64(destAmount), ledgerCloseTime, now)
}
