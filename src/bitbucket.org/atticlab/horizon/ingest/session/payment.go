package session

import (
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/log"
	"database/sql"
)

func (is *Session) ingestPayment(sourceAddress, destAddress string, sourceAmount, destAmount xdr.Int64, sourceAsset, destAsset string) error {

	sourceAccount, err := is.Ingestion.HistoryAccountCache.Get(sourceAddress)
	if err != nil {
		return err
	}

	destAccount, err := is.Ingestion.HistoryAccountCache.Get(destAddress)
	isDestNew := false
	if err != nil {
		if err != sql.ErrNoRows {
			return err
		}
		isDestNew = true
	}

	if isDestNew {
		destAccount = history.NewAccount(is.Cursor.OperationID(), destAddress, xdr.AccountTypeAccountAnonymousUser)
		err = is.Ingestion.Account(destAccount, true, &destAsset, &sourceAccount.AccountType)
		if err != nil {
			log.Error("Failed to ingest anonymous account created by payment!")
			return err
		}
	}

	return nil
}
