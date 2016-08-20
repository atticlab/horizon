package ingestion

import (
	"database/sql"
	"time"

	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/db2/history"
)

// UpdateAccountIncome updates income stats for specified account and asset
func (ingest *Ingestion) UpdateAccountIncome(address string, assetCode string, counterpartyType xdr.AccountType,
	income int64, ledgerClosedAt time.Time, now time.Time) error {
	return ingest.updateAccountStats(address, assetCode, counterpartyType, income, ledgerClosedAt, now, true)
}

// UpdateAccountOutcome updates outcome stats for specified account and asset
func (ingest *Ingestion) UpdateAccountOutcome(address string, assetCode string, counterpartyType xdr.AccountType, outcome int64,
	ledgerClosedAt time.Time, now time.Time) error {
	return ingest.updateAccountStats(address, assetCode, counterpartyType, outcome, ledgerClosedAt, now, false)
}

// updateAccountStats updates outcome stats for specified account and asset
func (ingest *Ingestion) updateAccountStats(address string, assetCode string, counterpartyType xdr.AccountType,
	delta int64, //account balance change
	ledgerClosedAt time.Time, now time.Time,
	income bool, // payment direction
) error {
	isNew := false
	stats, err := ingest.accountStatsCache.Get(address, assetCode, counterpartyType)
	if err != nil {
		if err != sql.ErrNoRows {
			return err
		}
		isNew = true
		rawStats := history.NewAccountStatistics(address, assetCode, counterpartyType)
		stats = &rawStats
		ingest.accountStatsCache.Add(stats)
	} else {
		stats.ClearObsoleteStats(now)
	}

	stats.Update(delta, ledgerClosedAt, now, income)
	stats.UpdatedAt = now

	if isNew {
		err = ingest.historyQ.CreateAccountStats(stats)
	} else {
		err = ingest.historyQ.UpdateAccountStats(stats)
	}
	return err
}
