package history

import (
    "time"
    "fmt"
    "bitbucket.org/atticlab/horizon/helpers"
    "bitbucket.org/atticlab/horizon/log"
	sq "github.com/lann/squirrel"
    dbsql "database/sql"
)

// StatisticsByAccountAndAsset loads a row from `account_statistics`, by address and asset code
func (q *Q) StatisticsByAccountAndAsset(dest *AccountStatistics, addy string, assetCode string) error {
	sql := SelectAccountStatisticsTemplate.Where("a.address = ? AND a.asset_code = ?", addy, assetCode)
    var stats AccountStatistics
	err := q.Get(&stats, sql)
    
    now := time.Now()
    if err == dbsql.ErrNoRows {
        // Construct fake account statistics entry
        stats.Account = addy
        stats.AssetCode = assetCode
        stats.UpdatedAt = now
        *dest = stats
        
        return nil
    } else if err == nil {
        // Erase obsolete data from result. Don't save, to avoid conflicts with ingester's thread
        stats.ClearObsoleteStats(now)
        *dest = stats
    }
    
    return err
}

// ClearObsoleteStats checks last update time and erases obsolete data
func (stats *AccountStatistics) ClearObsoleteStats(now time.Time) {
    if stats.UpdatedAt.Year() < now.Year() {
        
        log.Info(
            fmt.Sprintf(
                "account_statistics: Ereasing obsolete stats for %s - %s (YEAR).",
                stats.Account,
                stats.AssetCode,
        ))
        
        stats.AnnualIncome = 0
        stats.AnnualOutcome = 0
        stats.MonthlyIncome = 0
        stats.MonthlyOutcome = 0
        stats.WeeklyIncome = 0
        stats.WeeklyOutcome = 0
        stats.DailyIncome = 0
        stats.DailyOutcome = 0
        
        stats.UpdatedAt = now
    } else if stats.UpdatedAt.Month() < now.Month() {
        
        log.Info(
            fmt.Sprintf(
                "account_statistics: Ereasing obsolete stats for %s - %s (MONTH).",
                stats.Account,
                stats.AssetCode,
        ))
        
        stats.MonthlyIncome = 0
        stats.MonthlyOutcome = 0
        stats.WeeklyIncome = 0
        stats.WeeklyOutcome = 0
        stats.DailyIncome = 0
        stats.DailyOutcome = 0
        
        stats.UpdatedAt = now
    } else if !helpers.SameWeek(stats.UpdatedAt, now) {
        
        log.Info(
            fmt.Sprintf(
                "account_statistics: Ereasing obsolete stats for %s - %s (WEEK).",
                stats.Account,
                stats.AssetCode,
        ))
        
        stats.WeeklyIncome = 0
        stats.WeeklyOutcome = 0
        stats.DailyIncome = 0
        stats.DailyOutcome = 0
        
        stats.UpdatedAt = now
    } else if stats.UpdatedAt.Day() < now.Day() {
        
        log.Info(
            fmt.Sprintf(
                "account_statistics: Ereasing obsolete stats for %s - %s (DAY).",
                stats.Account,
                stats.AssetCode,
        ))
        
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


