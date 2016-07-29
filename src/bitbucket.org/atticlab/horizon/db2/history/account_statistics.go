package history

import (
	"time"

	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/helpers"
	"bitbucket.org/atticlab/horizon/log"
	sq "github.com/lann/squirrel"
)

// GetAccountStatistics returns account_statistics row by account, asset and counterparty type.
func (q *Q) GetAccountStatistics(
	dest *AccountStatistics,
	address string,
	assetCode string,
	counterPartyType xdr.AccountType,
) error {
	sql := SelectAccountStatisticsTemplate.Where(
		"a.address = ? AND a.asset_code = ? AND a.counterparty_type = ?",
		address,
		assetCode,
		int16(counterPartyType),
	)

	now := time.Now()
	var stats AccountStatistics
	err := q.Get(&stats, sql)

	if err == nil {
		// Erase obsolete data from result. Don't save, to avoid conflicts with ingester's thread
		stats.ClearObsoleteStats(now)
		*dest = stats
	}

	return err
}

// GetStatisticsByAccount selects rows from `account_statistics` by address
func (q *Q) GetStatisticsByAccount(dest *[]AccountStatistics, addy string) error {
	sql := SelectAccountStatisticsTemplate.Where("a.address = ?", addy)
	var stats []AccountStatistics
	err := q.Select(&stats, sql)

	if err == nil {
		now := time.Now()
		for _, stat := range stats {
			// Erase obsolete data from result. Don't save, to avoid conflicts with ingester's thread
			stat.ClearObsoleteStats(now)
		}
		*dest = stats
	}

	return err
}

func (q *Q) CreateStats(stats AccountStatistics) error {
	sql := CreateAccountStatisticsTemplate.Values(
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

	_, err := q.Exec(sql)
	return err
}

// GetStatisticsByAccountAndAsset selects rows from `account_statistics` by address and asset code
func (q *Q) GetStatisticsByAccountAndAsset(dest map[xdr.AccountType]AccountStatistics, addy string, assetCode string) error {
	sql := SelectAccountStatisticsTemplate.Where("a.address = ? AND a.asset_code = ?", addy, assetCode)
	var stats []AccountStatistics
	err := q.Select(&stats, sql)
	if err != nil {
		return err
	}
	now := time.Now()
	for _, stat := range stats {
		// Erase obsolete data from result. Don't save, to avoid conflicts with ingester's thread
		stat.ClearObsoleteStats(now)
		dest[xdr.AccountType(stat.CounterpartyType)] = stat
	}

	return nil
}

// ClearObsoleteStats checks last update time and erases obsolete data
func (stats *AccountStatistics) ClearObsoleteStats(now time.Time) {
	isYear := stats.UpdatedAt.Year() < now.Year()
	if isYear {
		stats.AnnualIncome = 0
		stats.AnnualOutcome = 0
	}
	isMonth := isYear || stats.UpdatedAt.Month() < now.Month()
	if isMonth {

		stats.MonthlyIncome = 0
		stats.MonthlyOutcome = 0
	}
	isWeek := isMonth || !helpers.SameWeek(stats.UpdatedAt, now)
	if isWeek {
		stats.WeeklyIncome = 0
		stats.WeeklyOutcome = 0
	}
	isDay := isWeek || stats.UpdatedAt.Day() < now.Day()
	if isDay {

		log.WithFields(
			log.F{
				"account":           stats.Account,
				"asset":             stats.AssetCode,
				"counterparty_type": stats.CounterpartyType,
				"year":              isYear,
				"month":             isMonth,
				"week":              isWeek,
				"day":               isDay,
			}).Info("account_statistics: Ereasing obsolete stats")

		stats.DailyIncome = 0
		stats.DailyOutcome = 0

		stats.UpdatedAt = now
	}
}

// TODO: get all assets for account

// SelectAccountStatisticsTemplate is a prepared statement for SELECT from the account_statistics
var SelectAccountStatisticsTemplate = sq.Select("a.*").From("account_statistics a")

// CreateAccountStatisticsTemplate is a prepared statement for insertion into the account_statistics
var CreateAccountStatisticsTemplate = sq.Insert("account_statistics").Columns(
	"address",
	"asset_code",
	"counterparty_type",
	"daily_income",
	"daily_outcome",
	"weekly_income",
	"weekly_outcome",
	"monthly_income",
	"monthly_outcome",
	"annual_income",
	"annual_outcome",
	"updated_at",
)
