// Package ingest contains the ingestion system for horizon.  This system takes
// data produced by the connected stellar-core database, transforms it and
// inserts it into the horizon database.
package ingest

import (
	"time"

	"bitbucket.org/atticlab/horizon/db2"
	"bitbucket.org/atticlab/horizon/ingest/session"
)

const (
	// CurrentVersion reflects the latest version of the ingestion
	// algorithm. As rows are ingested into the horizon database, this version is
	// used to tag them.  In the future, any breaking changes introduced by a
	// developer should be accompanied by an increase in this value.
	//
	// Scripts, that have yet to be ported to this codebase can then be leveraged
	// to re-ingest old data with the new algorithm, providing a seamless
	// transition when the ingested data's structure changes.
	CurrentVersion = 8
)

// System represents the data ingestion subsystem of horizon.
type System struct {
	// HorizonDB is the connection to the horizon database that ingested data will
	// be written to.
	HorizonDB *db2.Repo

	// CoreDB is the stellar-core db that data is ingested from.
	CoreDB *db2.Repo

	Metrics *session.IngesterMetrics

	// Network is the passphrase for the network being imported
	Network string

	tick            *time.Ticker
	historySequence int32
	coreSequence    int32
}

// New initializes the ingester, causing it to begin polling the stellar-core
// database for now ledgers and ingesting data into the horizon database.
func New(network string, core, horizon *db2.Repo) *System {
	return &System{
		Network:   network,
		HorizonDB: horizon,
		CoreDB:    core,
		Metrics:   session.NewMetrics(),
		tick:      time.NewTicker(1 * time.Second),
	}
}

// ReingestAll re-ingests all data
func ReingestAll(network string, core, horizon *db2.Repo) (int, error) {
	i := New(network, core, horizon)
	return i.ReingestAll()
}

func ReingestOutdated(network string, core, horizon *db2.Repo) (int, error) {
	i := New(network, core, horizon)
	return i.ReingestOutdated()
}

// ReingestSingle re-ingests a single ledger
func ReingestSingle(network string, core, horizon *db2.Repo, sequence int32) error {
	i := New(network, core, horizon)
	return i.ReingestSingle(sequence)
}

// RunOnce runs a single ingestion session
func RunOnce(network string, core, horizon *db2.Repo) (*session.Session, error) {
	i := New(network, core, horizon)
	err := i.updateLedgerState()
	if err != nil {
		return nil, err
	}

	is := session.NewSession(
		i.historySequence+1,
		i.coreSequence,
		i.HorizonDB,
		i.CoreDB,
		i.Metrics,
		CurrentVersion,
	)

	is.Run()

	return is, is.Err
}
