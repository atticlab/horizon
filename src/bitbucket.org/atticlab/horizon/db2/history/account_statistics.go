package history

import (
	"time"

	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/helpers"
	"bitbucket.org/atticlab/horizon/log"
	sq "github.com/lann/squirrel"
)

func NewAccountStatistics(account, assetCode string, counterparty xdr.AccountType) AccountStatistics {
	return AccountStatistics{
		Account:          account,
		AssetCode:        assetCode,
		CounterpartyType: int16(counterparty),
	}
}

func (stats *AccountStatistics) Update(delta int64, receivedAt time.Time, now time.Time, isIncome bool) {
	log.WithFields(log.F{
		"service":    "account_statistics",
		"delta":      delta,
		"receivedAt": receivedAt,
		"now":        now,
		"isIncome":   isIncome,
	}).Debug("Updating")
	if isIncome {
		stats.AddIncome(delta, receivedAt, now)
	} else {
		stats.AddOutcome(delta, receivedAt, now)
	}
}

func (stats *AccountStatistics) AddIncome(income int64, receivedAt time.Time, now time.Time) {
	if receivedAt.Year() != now.Year() {
		return
	}
	stats.AnnualIncome = stats.AnnualIncome + income

	if receivedAt.Month() != now.Month() {
		return
	}
	stats.MonthlyIncome = stats.MonthlyIncome + income

	if !helpers.SameWeek(receivedAt, now) {
		return
	}
	stats.WeeklyIncome = stats.WeeklyIncome + income

	if receivedAt.Day() != now.Day() {
		return
	}
	stats.DailyIncome = stats.DailyIncome + income
}

func (stats *AccountStatistics) AddOutcome(outcome int64, performedAt time.Time, now time.Time) {
	if performedAt.Year() != now.Year() {
		return
	}
	stats.AnnualOutcome = stats.AnnualOutcome + outcome

	if performedAt.Month() != now.Month() {
		return
	}
	stats.MonthlyOutcome = stats.MonthlyOutcome + outcome

	if !helpers.SameWeek(performedAt, now) {
		return
	}
	stats.WeeklyOutcome = stats.WeeklyOutcome + outcome

	if performedAt.Day() != now.Day() {
		return
	}
	stats.DailyOutcome = stats.DailyOutcome + outcome
}

// GetAccountStatistics returns account_statistics row by account, asset and counterparty type.
func (q *Q) GetAccountStatistics(address string, assetCode string, counterPartyType xdr.AccountType) (AccountStatistics, error) {
	sql := selectAccountStatisticsTemplate.Limit(1).Where("a.address = ? AND a.asset_code = ? AND a.counterparty_type = ?",
		address,
		assetCode,
		int16(counterPartyType),
	)

	var stats AccountStatistics
	err := q.Get(&stats, sql)
	return stats, err
}

// GetStatisticsByAccount selects rows from `account_statistics` by address
func (q *Q) GetStatisticsByAccount(dest *[]AccountStatistics, addy string) error {
	sql := selectAccountStatisticsTemplate.Where("a.address = ?", addy)
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

// CreateAccountStats creates new row in the account_statistics table
// and populates it with values from the AccountStatistics struct
func (q *Q) CreateAccountStats(stats AccountStatistics) error {
	sql := createAccountStatisticsTemplate.Values(
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
func (q *Q) GetStatisticsByAccountAndAsset(dest map[xdr.AccountType]AccountStatistics, addy string, assetCode string, now time.Time) error {
	sql := selectAccountStatisticsTemplate.Where("a.address = ? AND a.asset_code = ?", addy, assetCode)
	var stats []AccountStatistics
	err := q.Select(&stats, sql)
	if err != nil {
		return err
	}

	for _, stat := range stats {
		// Erase obsolete data from result. Don't save, to avoid conflicts with ingester's thread
		stat.ClearObsoleteStats(now)
		dest[xdr.AccountType(stat.CounterpartyType)] = stat
	}

	return nil
}

// updateStats updates entry in the account_statistics table
// with values from the AccountStatistics struct
func (q *Q) UpdateAccountStats(stats AccountStatistics) error {
	update := updateAccountStatisticsTemplate.SetMap(map[string]interface{}{
		"daily_income":    stats.DailyIncome,
		"daily_outcome":   stats.DailyOutcome,
		"weekly_income":   stats.WeeklyIncome,
		"weekly_outcome":  stats.WeeklyOutcome,
		"monthly_income":  stats.MonthlyIncome,
		"monthly_outcome": stats.MonthlyOutcome,
		"annual_income":   stats.AnnualIncome,
		"annual_outcome":  stats.AnnualOutcome,
		"updated_at":      stats.UpdatedAt,
	}).Where(
		"address = ? AND asset_code = ? AND counterparty_type = ?",
		stats.Account,
		stats.AssetCode,
		stats.CounterpartyType,
	)

	_, err := q.Exec(update)
	return err
}

// ClearObsoleteStats checks last update time and erases obsolete data
func (stats *AccountStatistics) ClearObsoleteStats(now time.Time) {
	log.WithField("now", now).WithField("updated_at", stats.UpdatedAt).Debug("Clearing obsolete")
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
	log.WithFields(
		log.F{
			"service":           "account_statistics",
			"account":           stats.Account,
			"asset":             stats.AssetCode,
			"counterparty_type": stats.CounterpartyType,
			"year":              isYear,
			"month":             isMonth,
			"week":              isWeek,
			"day":               isDay,
			"now":               now.String(),
			"updated":           stats.UpdatedAt.String(),
		}).Debug("Erasing obsolete stats")
	if isDay {
		stats.DailyIncome = 0
		stats.DailyOutcome = 0

		stats.UpdatedAt = now
	}
}

// TODO: get all assets for account

// SelectAccountStatisticsTemplate is a prepared statement for SELECT from the account_statistics
var selectAccountStatisticsTemplate = sq.Select("a.*").From("account_statistics a")

// CreateAccountStatisticsTemplate is a prepared statement for insertion into the account_statistics
var createAccountStatisticsTemplate = sq.Insert("account_statistics").Columns(
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

var updateAccountStatisticsTemplate = sq.Update("account_statistics")
