package ingestion

import (
	"bitbucket.org/atticlab/horizon/cache"
	"bitbucket.org/atticlab/horizon/db2"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/db2/sqx"
	sq "github.com/lann/squirrel"
)

// Ingestion receives write requests from a Session
type Ingestion struct {
	// DB is the sql repo to be used for writing any rows into the horizon
	// database.
	DB                       *db2.Repo
	CurrentVersion           int

	ledgers                  sq.InsertBuilder
	transactions             sq.InsertBuilder
	transaction_participants sq.InsertBuilder
	operations               sq.InsertBuilder
	operation_participants   sq.InsertBuilder
	effects                  sq.InsertBuilder
	accounts                 sq.InsertBuilder

	statistics               *sqx.BatchUpdateBuilder

	// cache
	statisticsCache          *cache.AccountStatistics
	HistoryAccountCache      *cache.HistoryAccountID
}

func New(db *db2.Repo, accountStatsCache *cache.AccountStatistics, currentVersion int) *Ingestion {
	q := &history.Q{
		Repo: db,
	}
	return &Ingestion{
		DB:                  db,
		CurrentVersion:      currentVersion,
		HistoryAccountCache: cache.NewHistoryAccount(q),
		statisticsCache:     cache.NewAccountStatistics(q),
	}
}

// Rollback aborts this ingestions transaction
func (ingest *Ingestion) Rollback() (err error) {
	// recreates all inserters to release memory
	ingest.createInsertBuilders()
	err = ingest.DB.Rollback()
	return
}

// Start makes the ingestion reeady, initializing the insert builders and tx
func (ingest *Ingestion) Start() (err error) {
	err = ingest.DB.Begin()
	if err != nil {
		return
	}

	ingest.createInsertBuilders()

	return
}

// Clear removes data from the ledger
func (ingest *Ingestion) Clear(start int64, end int64) error {

	if start <= 1 {
		del := sq.Delete("history_accounts").Where("id = 1")
		ingest.DB.Exec(del)
	}

	err := ingest.clearRange(start, end, "history_effects", "history_operation_id")
	if err != nil {
		return err
	}
	err = ingest.clearRange(start, end, "history_operation_participants", "history_operation_id")
	if err != nil {
		return err
	}
	err = ingest.clearRange(start, end, "history_operations", "id")
	if err != nil {
		return err
	}
	err = ingest.clearRange(start, end, "history_transaction_participants", "history_transaction_id")
	if err != nil {
		return err
	}
	err = ingest.clearRange(start, end, "history_transactions", "id")
	if err != nil {
		return err
	}
	err = ingest.clearRange(start, end, "history_accounts", "id")
	if err != nil {
		return err
	}
	err = ingest.clearRange(start, end, "history_ledgers", "id")
	if err != nil {
		return err
	}

	return nil
}

// Close finishes the current transaction and finishes this ingestion.
func (ingest *Ingestion) Close() error {
	err := ingest.flushInserters()
	if err != nil {
		return err
	}
	return ingest.commit()
}

// Flush writes the currently buffered rows to the db, and if successful
// starts a new transaction.
func (ingest *Ingestion) Flush() error {
	err := ingest.flushInserters()
	if err != nil {
		return err
	}
	err = ingest.commit()
	if err != nil {
		return err
	}

	return ingest.Start()
}

func (ingest *Ingestion) flushInserters() error {
	err := ingest.statistics.Flush()
	if err != nil {
		return err
	}
	return nil
}

func (ingest *Ingestion) createInsertBuilders() {
	ingest.statistics = sqx.BatchUpdate(sqx.BatchInsertFromInsert(ingest.DB, history.AccountStatisticsCreate),
		history.AccountStatisticsUpdateParams, history.AccountStatisticsUpdateWhere)

	ingest.ledgers = sq.Insert("history_ledgers").Columns(
		"importer_version",
		"id",
		"sequence",
		"ledger_hash",
		"previous_ledger_hash",
		"total_coins",
		"fee_pool",
		"base_fee",
		"base_reserve",
		"max_tx_set_size",
		"closed_at",
		"created_at",
		"updated_at",
		"transaction_count",
		"operation_count",
	)

	ingest.accounts = sq.Insert("history_accounts").Columns(
		"id",
		"address",
	)

	ingest.transactions = sq.Insert("history_transactions").Columns(
		"id",
		"transaction_hash",
		"ledger_sequence",
		"application_order",
		"account",
		"account_sequence",
		"fee_paid",
		"operation_count",
		"tx_envelope",
		"tx_result",
		"tx_meta",
		"tx_fee_meta",
		"signatures",
		"time_bounds",
		"memo_type",
		"memo",
		"created_at",
		"updated_at",
	)

	ingest.transaction_participants = sq.Insert("history_transaction_participants").Columns(
		"history_transaction_id",
		"history_account_id",
	)

	ingest.operations = sq.Insert("history_operations").Columns(
		"id",
		"transaction_id",
		"application_order",
		"source_account",
		"type",
		"details",
	)

	ingest.operation_participants = sq.Insert("history_operation_participants").Columns(
		"history_operation_id",
		"history_account_id",
	)

	ingest.effects = sq.Insert("history_effects").Columns(
		"history_account_id",
		"history_operation_id",
		"\"order\"",
		"type",
		"details",
	)
}
