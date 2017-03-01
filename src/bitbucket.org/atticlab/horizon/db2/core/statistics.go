package core

type AccountStatistics struct {
	AccountID    string `db:"account_id"`
	AssetIssuer  string `db:"asset_issuer"`
	AssetCode    string `db:"asset_code"`
	AssetType    int    `db:"asset_type"`
	Counterparty int    `db:"counterparty"`
	DailyIn      int64  `db:"daily_in"`
	DailyOut     int64  `db:"daily_out"`
	MonthlyIn    int64  `db:"monthly_in"`
	MonthlyOut   int64  `db:"monthly_out"`
	AnnualIn     int64  `db:"annual_in"`
	AnnualOut    int64  `db:"annual_out"`
	UpdatedAt    int64  `db:"updated_at"`
	LastModified int64  `db:"lastmodified"`
}
