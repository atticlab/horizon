package ingest

import (
	sql "database/sql"
	"time"

	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/helpers"
	sq "github.com/lann/squirrel"
)

// UpdateAccountIncome updates income stats for specified account and asset
func (ingest *Ingestion) UpdateAccountIncome(
	address string,
	assetCode string,
	counterpartyType xdr.AccountType,
	income int64,
	ledgerClosedAt time.Time,
	now time.Time,
) error {
	return ingest.updateAccountStats(address, assetCode, counterpartyType, income, ledgerClosedAt, now, true)
}

// UpdateAccountOutcome updates outcome stats for specified account and asset
func (ingest *Ingestion) UpdateAccountOutcome(
	address string,
	assetCode string,
	counterpartyType xdr.AccountType,
	outcome int64,
	ledgerClosedAt time.Time,
	now time.Time,
) error {
	return ingest.updateAccountStats(address, assetCode, counterpartyType, outcome, ledgerClosedAt, now, false)
}

// updateAccountStats updates outcome stats for specified account and asset
func (ingest *Ingestion) updateAccountStats(
	address string,
	assetCode string,
	counterpartyType xdr.AccountType,
	delta int64, //account balance change
	ledgerClosedAt time.Time,
	now time.Time,
	income bool, // payment direction
) error {
	stats, err := ingest.getStats(address, assetCode, counterpartyType)

	if err == nil {
		// Update account statistics
		stats.ClearObsoleteStats(now)

		if income {
			stats = updateIncome(stats, delta, ledgerClosedAt, now)
		} else {
			stats = updateOutcome(stats, delta, ledgerClosedAt, now)
		}
		stats.UpdatedAt = helpers.MaxDate(stats.UpdatedAt, ledgerClosedAt)

		err = ingest.updateStats(stats)
	} else if err == sql.ErrNoRows {
		// Create new account statistics entry
		stats.Account = address
		stats.AssetCode = assetCode
		stats.CounterpartyType = int16(counterpartyType)

		if income {
			stats = updateIncome(stats, delta, ledgerClosedAt, now)
		} else {
			stats = updateOutcome(stats, delta, ledgerClosedAt, now)
		}
		stats.UpdatedAt = ledgerClosedAt

		err = ingest.createStats(stats)
	}

	return err
}

func updateIncome(stats history.AccountStatistics, delta int64, timestamp time.Time, now time.Time) history.AccountStatistics {
	if timestamp.Year() == now.Year() {
		stats.AnnualIncome = stats.AnnualIncome + delta
		if timestamp.Month() == now.Month() {
			stats.MonthlyIncome = stats.MonthlyIncome + delta
			if helpers.SameWeek(timestamp, now) {
				stats.WeeklyIncome = stats.WeeklyIncome + delta
				if timestamp.Day() == now.Day() {
					stats.DailyIncome = stats.DailyIncome + delta
				}
			}
		}
	}

	return stats
}

func updateOutcome(stats history.AccountStatistics, delta int64, timestamp time.Time, now time.Time) history.AccountStatistics {

	if timestamp.Year() == now.Year() {
		stats.AnnualOutcome = stats.AnnualOutcome + delta
		if timestamp.Month() == now.Month() {
			stats.MonthlyOutcome = stats.MonthlyOutcome + delta
			if helpers.SameWeek(timestamp, now) {
				stats.WeeklyOutcome = stats.WeeklyOutcome + delta
				if timestamp.Day() == now.Day() {
					stats.DailyOutcome = stats.DailyOutcome + delta
				}
			}
		}
	}

	return stats
}

func (ingest *Ingestion) getStats(
	address string,
	assetCode string,
	counterpartyType xdr.AccountType,
) (history.AccountStatistics, error) {
	var stats history.AccountStatistics
	sql := history.SelectAccountStatisticsTemplate.Limit(1).Where(
		"a.address = ? AND a.asset_code = ? AND a.counterparty_type = ?",
		address,
		assetCode,
		int16(counterpartyType),
	)
	err := ingest.DB.Get(&stats, sql)

	return stats, err
}

// createStats creates new row in the account_statistics table
// and populates it with values from the AccountStatistics struct
func (ingest *Ingestion) createStats(stats history.AccountStatistics) error {
	sql := history.CreateAccountStatisticsTemplate.Values(
		stats.Account,
		stats.AssetCode,
		int16(stats.CounterpartyType),
		stats.DailyIncome,
		stats.DailyOutcome,
		stats.WeeklyIncome,
		stats.WeeklyOutcome,
		stats.MonthlyIncome,
		stats.MonthlyOutcome,
		stats.AnnualIncome,
		stats.AnnualOutcome,
		stats.UpdatedAt,
	)

	_, err := ingest.DB.Exec(sql)

	return err
}

// updateStats updates entry in the account_statistics table
// with values from the AccountStatistics struct
func (ingest *Ingestion) updateStats(stats history.AccountStatistics) error {
	sql := sq.Update("account_statistics")
	sql = sql.Set("daily_income", stats.DailyIncome)
	sql = sql.Set("daily_outcome", stats.DailyOutcome)
	sql = sql.Set("weekly_income", stats.WeeklyIncome)
	sql = sql.Set("weekly_outcome", stats.WeeklyOutcome)
	sql = sql.Set("monthly_income", stats.MonthlyIncome)
	sql = sql.Set("monthly_outcome", stats.MonthlyOutcome)
	sql = sql.Set("annual_income", stats.AnnualIncome)
	sql = sql.Set("annual_outcome", stats.AnnualOutcome)
	sql = sql.Set("updated_at", stats.UpdatedAt)
	sql = sql.Where(
		"address = ? AND asset_code = ? AND counterparty_type = ?",
		stats.Account,
		stats.AssetCode,
		stats.CounterpartyType,
	)

	_, err := ingest.DB.Exec(sql)

	return err
}
