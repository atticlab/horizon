package ingestion

import (
	"bitbucket.org/atticlab/horizon/cache"
	"bitbucket.org/atticlab/horizon/db2"
	sq "github.com/lann/squirrel"
)

// Ingestion receives write requests from a Session
type Ingestion struct {
	// DB is the sql repo to be used for writing any rows into the horizon
	// database.
	DB             *db2.Repo
	accountCache   *cache.HistoryAccountID
	CurrentVersion int

	ledgers                  sq.InsertBuilder
	transactions             sq.InsertBuilder
	transaction_participants sq.InsertBuilder
	operations               sq.InsertBuilder
	operation_participants   sq.InsertBuilder
	effects                  sq.InsertBuilder
	accounts                 sq.InsertBuilder
}

func New(db *db2.Repo, accountCache *cache.HistoryAccountID, currentVersion int) *Ingestion {
	return &Ingestion{
		DB:             db,
		accountCache:   accountCache,
		CurrentVersion: currentVersion,
	}
}
