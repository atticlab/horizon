package session

import (
	"bitbucket.org/atticlab/horizon/cache"
	"bitbucket.org/atticlab/horizon/db2"
	"bitbucket.org/atticlab/horizon/db2/core"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/ingest/session/ingestion"
)

// Session represents a single attempt at ingesting data into the history
// database.
type Session struct {
	Cursor    *Cursor
	Ingestion *ingestion.Ingestion

	// ClearExisting causes the session to clear existing data from the horizon db
	// when the session is run.
	ClearExisting bool

	// Metrics is a reference to where the session should record its metric information
	Metrics *IngesterMetrics

	accountIDCache   *cache.HistoryAccountID
	accountTypeCache *cache.AccountType

	//
	// Results fields
	//

	// Ingested is the number of ledgers that were successfully ingested during
	// this session.
	Ingested int
}

// NewSession initialize a new ingestion session, from `first` to `last`
func NewSession(first, last int32, horizonDB *db2.Repo, coreDB *db2.Repo, metrics *IngesterMetrics, currentVersion int) *Session {
	hdb := horizonDB.Clone()

	historyQ := &history.Q{
		Repo: hdb,
	}
	accountIdCache := cache.NewHistoryAccount(historyQ)
	return &Session{
		Ingestion:      ingestion.New(hdb, accountIdCache, currentVersion),
		Cursor:         NewCursor(coreDB, first, last, metrics.LoadLedgerTimer),
		Metrics:        metrics,
		accountIDCache: accountIdCache,
		accountTypeCache: cache.NewAccountType(&core.Q{
			Repo: coreDB,
		}),
	}
}
