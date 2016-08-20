package session

import (
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/log"
	"time"
)

func (is *Session) ingestPayment(from, to string, sourceAmount, destAmount xdr.Int64, sourceAsset, destAsset string) error {

	sourceType, err := is.accountTypeCache.Get(from)
	if err != nil {
		return err
	}

	destinationType, err := is.accountTypeCache.Get(to)
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
	now := time.Now()
	err = is.Ingestion.UpdateAccountOutcome(from, sourceAsset, destinationType, int64(sourceAmount), ledgerCloseTime, now)
	if err != nil {
		return err
	}

	return is.Ingestion.UpdateAccountIncome(to, destAsset, sourceType, int64(destAmount), ledgerCloseTime, now)
}
