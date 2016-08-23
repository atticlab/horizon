package ingestion

import (
	"encoding/json"
	"fmt"
	"time"

	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/db2/core"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/db2/sqx"
	"github.com/guregu/null"
	sq "github.com/lann/squirrel"
	"database/sql"
)

// Account ingests the provided account data into a new row in the
// `history_accounts` table
func (ingest *Ingestion) Account(id int64, address string) error {

	_, err := ingest.HistoryAccountCache.Get(address)

	if err != sql.ErrNoRows {
		return err
	}

	sql := ingest.accounts.Values(id, address)

	_, err = ingest.DB.Exec(sql)
	if err != nil {
		return err
	}

	ingest.HistoryAccountCache.Add(address, id)

	return nil
}

// Effect adds a new row into the `history_effects` table.
func (ingest *Ingestion) Effect(aid int64, opid int64, order int, typ history.EffectType, details interface{}) error {
	djson, err := json.Marshal(details)
	if err != nil {
		return err
	}

	sql := ingest.effects.Values(aid, opid, order, typ, djson)

	_, err = ingest.DB.Exec(sql)
	if err != nil {
		return err
	}

	return nil
}

// Ledger adds a ledger to the current ingestion
func (ingest *Ingestion) Ledger(
	id int64,
	header *core.LedgerHeader,
	txs int,
	ops int,
) error {

	sql := ingest.ledgers.Values(
		ingest.CurrentVersion,
		id,
		header.Sequence,
		header.LedgerHash,
		null.NewString(header.PrevHash, header.Sequence > 1),
		header.Data.TotalCoins,
		header.Data.FeePool,
		header.Data.BaseFee,
		header.Data.BaseReserve,
		header.Data.MaxTxSetSize,
		time.Unix(header.CloseTime, 0).UTC(),
		time.Now().UTC(),
		time.Now().UTC(),
		txs,
		ops,
	)

	_, err := ingest.DB.Exec(sql)
	if err != nil {
		return err
	}

	return nil
}

// Operation ingests the provided operation data into a new row in the
// `history_operations` table
func (ingest *Ingestion) Operation(
	id int64,
	txid int64,
	order int32,
	source xdr.AccountId,
	typ xdr.OperationType,
	details map[string]interface{},

) error {
	djson, err := json.Marshal(details)
	if err != nil {
		return err
	}

	sql := ingest.operations.Values(id, txid, order, source.Address(), typ, djson)
	_, err = ingest.DB.Exec(sql)
	if err != nil {
		return err
	}

	return nil
}

// OperationParticipants ingests the provided accounts `aids` as participants of
// operation with id `op`, creating a new row in the
// `history_operation_participants` table.
func (ingest *Ingestion) OperationParticipants(op int64, aids []int64) error {
	sql := ingest.operation_participants

	for _, aid := range aids {
		sql = sql.Values(op, aid)
	}

	_, err := ingest.DB.Exec(sql)
	if err != nil {
		return err
	}

	return nil
}

// Transaction ingests the provided transaction data into a new row in the
// `history_transactions` table
func (ingest *Ingestion) Transaction(
	id int64,
	tx *core.Transaction,
	fee *core.TransactionFee,
) error {

	sql := ingest.transactions.Values(
		id,
		tx.TransactionHash,
		tx.LedgerSequence,
		tx.Index,
		tx.SourceAddress(),
		tx.Sequence(),
		tx.Fee(),
		len(tx.Envelope.Tx.Operations),
		tx.EnvelopeXDR(),
		tx.ResultXDR(),
		tx.ResultMetaXDR(),
		fee.ChangesXDR(),
		sqx.StringArray(tx.Base64Signatures()),
		formatTimeBounds(tx.Envelope.Tx.TimeBounds),
		tx.MemoType(),
		tx.Memo(),
		time.Now().UTC(),
		time.Now().UTC(),
	)

	_, err := ingest.DB.Exec(sql)
	if err != nil {
		return err
	}

	return nil
}

// TransactionParticipants ingests the provided account ids as participants of
// transaction with id `tx`, creating a new row in the
// `history_transaction_participants` table.
func (ingest *Ingestion) TransactionParticipants(tx int64, aids []int64) error {
	sql := ingest.transaction_participants

	for _, aid := range aids {
		sql = sql.Values(tx, aid)
	}

	_, err := ingest.DB.Exec(sql)
	if err != nil {
		return err
	}

	return nil
}

func (ingest *Ingestion) clearRange(start int64, end int64, table string, idCol string) error {
	del := sq.Delete(table).Where(
		fmt.Sprintf("%s >= ? AND %s < ?", idCol, idCol),
		start,
		end,
	)
	_, err := ingest.DB.Exec(del)
	return err
}

func (ingest *Ingestion) commit() error {
	err := ingest.DB.Commit()
	if err != nil {
		return err
	}

	return nil
}
