package history

import sq "github.com/lann/squirrel"

// GetAccountLimits returns limits row by account and asset.
func (q *Q) GetAccountLimits(
	dest *AccountLimits,
	address string,
	assetCode string,
) error {
	sql := SelectAccountLimitsTemplate.Where(
		"a.address = ? AND a.asset_code = ?",
		address,
		assetCode,
	)

	var limits AccountLimits
	err := q.Get(&limits, sql)

	if err == nil {
		*dest = limits
	}

	return err
}

// GetLimitsByAccount selects rows from `account_statistics` by address
func (q *Q) GetLimitsByAccount(dest *[]AccountLimits, address string) error {
	sql := SelectAccountLimitsTemplate.Where("a.address = ?", address)
	var limits []AccountLimits
	err := q.Select(&limits, sql)

	if err == nil {
		*dest = limits
	}

	return err
}

// CreateAccountLimits inserts new account_limits row
func (q *Q) CreateAccountLimits(limits AccountLimits) error {
	sql := CreateAccountLimitsTemplate.Values(limits.Account, limits.AssetCode, limits.MaxOperation, limits.DailyTurnover, limits.MonthlyTurnover)
	_, err := q.Exec(sql)

	return err
}

// UpdateAccountLimits updates account_limits row
func (q *Q) UpdateAccountLimits(limits AccountLimits) error {
	sql := UpdateAccountLimitsTemplate.Set("max_operation", limits.MaxOperation)
	sql = sql.Set("daily_turnover", limits.DailyTurnover)
	sql = sql.Set("monthly_turnover", limits.MonthlyTurnover)
	sql = sql.Where("address = ? and asset_code = ?", limits.Account, limits.AssetCode)

	_, err := q.Exec(sql)

	return err
}

// SelectAccountLimitsTemplate is a prepared statement for SELECT from the account_limits
var SelectAccountLimitsTemplate = sq.Select("a.*").From("account_limits a")

// CreateAccountLimitsTemplate is a prepared statement for insertion into the account_limits
var CreateAccountLimitsTemplate = sq.Insert("account_limits").Columns(
	"address",
	"asset_code",
	"max_operation",
	"daily_turnover",
	"monthly_turnover",
)

// UpdateAccountLimitsTemplate is a prepared statement for insertion into the account_limits
var UpdateAccountLimitsTemplate = sq.Update("account_limits")
