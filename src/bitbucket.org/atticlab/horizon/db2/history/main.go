// Package history contains database record definitions useable for
// reading rows from a the history portion of horizon's database
package history

import (
	"time"

	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/db2"
	"github.com/guregu/null"
	sq "github.com/lann/squirrel"
)

const (
	// account effects

	// EffectAccountCreated effects occur when a new account is created
	EffectAccountCreated EffectType = 0 // from create_account

	// EffectAccountRemoved effects occur when one account is merged into another
	EffectAccountRemoved EffectType = 1 // from merge_account

	// EffectAccountCredited effects occur when an account receives some currency
	EffectAccountCredited EffectType = 2 // from create_account, payment, path_payment, merge_account

	// EffectAccountDebited effects occur when an account sends some currency
	EffectAccountDebited EffectType = 3 // from create_account, payment, path_payment, create_account

	// EffectAccountThresholdsUpdated effects occur when an account changes its
	// multisig thresholds.
	EffectAccountThresholdsUpdated EffectType = 4 // from set_options

	// EffectAccountHomeDomainUpdated effects occur when an account changes its
	// home domain.
	EffectAccountHomeDomainUpdated EffectType = 5 // from set_options

	// EffectAccountFlagsUpdated effects occur when an account changes its
	// account flags, either clearing or setting.
	EffectAccountFlagsUpdated EffectType = 6 // from set_options

	// signer effects

	// EffectSignerCreated occurs when an account gains a signer
	EffectSignerCreated EffectType = 10 // from set_options

	// EffectSignerRemoved occurs when an account loses a signer
	EffectSignerRemoved EffectType = 11 // from set_options

	// EffectSignerUpdated occurs when an account changes the weight of one of its
	// signers.
	EffectSignerUpdated EffectType = 12 // from set_options

	// trustline effects

	// EffectTrustlineCreated occurs when an account trusts an anchor
	EffectTrustlineCreated EffectType = 20 // from change_trust

	// EffectTrustlineRemoved occurs when an account removes struct by setting the
	// limit of a trustline to 0
	EffectTrustlineRemoved EffectType = 21 // from change_trust

	// EffectTrustlineUpdated occurs when an account changes a trustline's limit
	EffectTrustlineUpdated EffectType = 22 // from change_trust, allow_trust

	// EffectTrustlineAuthorized occurs when an anchor has AUTH_REQUIRED flag set
	// to true and it authorizes another account's trustline
	EffectTrustlineAuthorized EffectType = 23 // from allow_trust

	// EffectTrustlineDeauthorized occurs when an anchor revokes access to a asset
	// it issues.
	EffectTrustlineDeauthorized EffectType = 24 // from allow_trust

	// trading effects

	// EffectOfferCreated occurs when an account offers to trade an asset
	EffectOfferCreated EffectType = 30 // from manage_offer, creat_passive_offer

	// EffectOfferRemoved occurs when an account removes an offer
	EffectOfferRemoved EffectType = 31 // from manage_offer, creat_passive_offer, path_payment

	// EffectOfferUpdated occurs when an offer is updated by the offering account.
	EffectOfferUpdated EffectType = 32 // from manage_offer, creat_passive_offer, path_payment

	// EffectTrade occurs when a trade is initiated because of a path payment or
	// offer operation.
	EffectTrade EffectType = 33 // from manage_offer, creat_passive_offer, path_payment

	// data effects

	// EffectDataCreated occurs when an account gets a new data field
	EffectDataCreated EffectType = 40 // from manage_data

	// EffectDataRemoved occurs when an account removes a data field
	EffectDataRemoved EffectType = 41 // from manage_data

	// EffectDataUpdated occurs when an account changes a data field's value
	EffectDataUpdated EffectType = 42 // from manage_data

	// EffectAdminOpPerformed occurs when an admin operation was performed
	EffectAdminOpPerformed EffectType = 43
)

// Account is a row of data from the `history_accounts` table
type Account struct {
	TotalOrderID
	Address string `db:"address"`
}

// AccountsQ is a helper struct to aid in configuring queries that loads
// slices of account structs.
type AccountsQ struct {
	Err    error
	parent *Q
	sql    sq.SelectBuilder
}

// Effect is a row of data from the `history_effects` table
type Effect struct {
	HistoryAccountID   int64       `db:"history_account_id"`
	Account            string      `db:"address"`
	HistoryOperationID int64       `db:"history_operation_id"`
	Order              int32       `db:"order"`
	Type               EffectType  `db:"type"`
	DetailsString      null.String `db:"details"`
}

// EffectsQ is a helper struct to aid in configuring queries that loads
// slices of Ledger structs.
type EffectsQ struct {
	Err    error
	parent *Q
	sql    sq.SelectBuilder
}

// EffectType is the numeric type for an effect, used as the `type` field in the
// `history_effects` table.
type EffectType int

// Ledger is a row of data from the `history_ledgers` table
type Ledger struct {
	TotalOrderID
	Sequence           int32       `db:"sequence"`
	ImporterVersion    int32       `db:"importer_version"`
	LedgerHash         string      `db:"ledger_hash"`
	PreviousLedgerHash null.String `db:"previous_ledger_hash"`
	TransactionCount   int32       `db:"transaction_count"`
	OperationCount     int32       `db:"operation_count"`
	ClosedAt           time.Time   `db:"closed_at"`
	CreatedAt          time.Time   `db:"created_at"`
	UpdatedAt          time.Time   `db:"updated_at"`
	TotalCoins         int64       `db:"total_coins"`
	FeePool            int64       `db:"fee_pool"`
	BaseFee            int32       `db:"base_fee"`
	BaseReserve        int32       `db:"base_reserve"`
	MaxTxSetSize       int32       `db:"max_tx_set_size"`
}

// LedgersQ is a helper struct to aid in configuring queries that loads
// slices of Ledger structs.
type LedgersQ struct {
	Err    error
	parent *Q
	sql    sq.SelectBuilder
}

// Operation is a row of data from the `history_operations` table
type Operation struct {
	TotalOrderID
	TransactionID    int64             `db:"transaction_id"`
	TransactionHash  string            `db:"transaction_hash"`
	ApplicationOrder int32             `db:"application_order"`
	Type             xdr.OperationType `db:"type"`
	DetailsString    null.String       `db:"details"`
	SourceAccount    string            `db:"source_account"`
	ClosedAt         time.Time         `db:"closed_at"`
}

// OperationsQ is a helper struct to aid in configuring queries that loads
// slices of Operation structs.
type OperationsQ struct {
	Err    error
	parent *Q
	sql    sq.SelectBuilder
}

// QInterface is a helper struct on which to hang common queries against a history
// portion of the horizon database.
type QInterface interface {
	// Account limits
	// GetAccountLimits returns limits row by account and asset.
	GetAccountLimits(dest interface{}, address string, assetCode string) error
	// Inserts new account limits instance
	CreateAccountLimits(limits AccountLimits) error
	// Updates account's limits
	UpdateAccountLimits(limits AccountLimits) error

	// Account statistics
	// GetStatisticsByAccountAndAsset selects rows from `account_statistics` by address and asset code
	// Now is used to clear obsolete stats
	GetStatisticsByAccountAndAsset(dest map[xdr.AccountType]AccountStatistics, addy string, assetCode string, now time.Time) error
	GetAccountStatistics(address string, assetCode string, counterPartyType xdr.AccountType) (AccountStatistics, error)
	// CreateAccountStats creates new row in the account_statistics table
	// and populates it with values from the AccountStatistics struct
	CreateAccountStats(stats *AccountStatistics) error
	// updateStats updates entry in the account_statistics table
	// with values from the AccountStatistics struct
	UpdateAccountStats(stats *AccountStatistics) error

	// Account traits
	// Returns query helper for account traits
	AccountTraitsQ() AccountTraitsQInterface
	// Inserts new instance of account traits
	InsertAccountTraits(traits AccountTraits) error
	// Updates account traits
	UpdateAccountTraits(traits AccountTraits) error
	// Deletes account traits by id
	DeleteAccountTraits(id int64) error

	// Asset
	// Returns asset for specified xdr.Asset
	Asset(dest interface{}, asset xdr.Asset) error
	// Deletes asset from db by id
	DeleteAsset(id int64) (bool, error)
	// updates asset
	UpdateAsset(asset *Asset) (bool, error)
	// inserts asset
	InsertAsset(asset *Asset) (err error)

	// Account
	// AccountByAddress loads a row from `history_accounts`, by address
	AccountByAddress(dest interface{}, addy string) error
	// loads a id from `history_accounts`, by address
	AccountIDByAddress(addy string) (int64, error)

	// Commission
	// selects commission by hash
	CommissionByHash(hash string) (*Commission, error)
	// Inserts new commission
	InsertCommission(commission *Commission) (err error)
	// Deletes commission
	DeleteCommission(hash string) (bool, error)
	// update commission
	UpdateCommission(commission *Commission) (bool, error)
}

// Q is default implementation of QInterface
type Q struct {
	*db2.Repo
}

// TotalOrderID represents the ID portion of rows that are identified by the
// "TotalOrderID".  See total_order_id.go in the `db` package for details.
type TotalOrderID struct {
	ID int64 `db:"id"`
}

// Transaction is a row of data from the `history_transactions` table
type Transaction struct {
	TotalOrderID
	TransactionHash  string      `db:"transaction_hash"`
	LedgerSequence   int32       `db:"ledger_sequence"`
	LedgerCloseTime  time.Time   `db:"ledger_close_time"`
	ApplicationOrder int32       `db:"application_order"`
	Account          string      `db:"account"`
	AccountSequence  string      `db:"account_sequence"`
	FeePaid          int32       `db:"fee_paid"`
	OperationCount   int32       `db:"operation_count"`
	TxEnvelope       string      `db:"tx_envelope"`
	TxResult         string      `db:"tx_result"`
	TxMeta           string      `db:"tx_meta"`
	TxFeeMeta        string      `db:"tx_fee_meta"`
	SignatureString  string      `db:"signatures"`
	MemoType         string      `db:"memo_type"`
	Memo             null.String `db:"memo"`
	ValidAfter       null.Int    `db:"valid_after"`
	ValidBefore      null.Int    `db:"valid_before"`
	CreatedAt        time.Time   `db:"created_at"`
	UpdatedAt        time.Time   `db:"updated_at"`
}

// TransactionsQ is a helper struct to aid in configuring queries that loads
// slices of transaction structs.
type TransactionsQ struct {
	Err    error
	parent *Q
	sql    sq.SelectBuilder
}

// LatestLedger loads the latest known ledger
func (q *Q) LatestLedger(dest interface{}) error {
	return q.GetRaw(dest, `SELECT COALESCE(MAX(sequence), 0) FROM history_ledgers`)
}

// OldestOutdatedLedgers populates a slice of ints with the first million
// outdated ledgers, based upon the provided `currentVersion` number
func (q *Q) OldestOutdatedLedgers(dest interface{}, currentVersion int) error {
	return q.SelectRaw(dest, `
		SELECT sequence
		FROM history_ledgers
		WHERE importer_version < $1
		ORDER BY sequence ASC
		LIMIT 1000000`, currentVersion)
}

// AccountStatistics is a row of data from the `account_statistics` table
type AccountStatistics struct {
	Account          string    `db:"address"`
	AssetCode        string    `db:"asset_code"`
	CounterpartyType int16     `db:"counterparty_type"`
	DailyIncome      int64     `db:"daily_income"`
	DailyOutcome     int64     `db:"daily_outcome"`
	WeeklyIncome     int64     `db:"weekly_income"`
	WeeklyOutcome    int64     `db:"weekly_outcome"`
	MonthlyIncome    int64     `db:"monthly_income"`
	MonthlyOutcome   int64     `db:"monthly_outcome"`
	AnnualIncome     int64     `db:"annual_income"`
	AnnualOutcome    int64     `db:"annual_outcome"`
	UpdatedAt        time.Time `db:"updated_at"`
}

// AccountStatisticsQ is a helper struct to aid in configuring queries that loads
// slices of Ledger structs.
type AccountStatisticsQ struct {
	Err    error
	parent *Q
	sql    sq.SelectBuilder
}

// AccountLimits contains limits for account set by the admin of a bank and
// is a row of data from the `account_limits` table
type AccountLimits struct {
	Account         string `db:"address"`
	AssetCode       string `db:"asset_code"`
	MaxOperationOut int64  `db:"max_operation_out"`
	DailyMaxOut     int64  `db:"daily_max_out"`
	MonthlyMaxOut   int64  `db:"monthly_max_out"`
	MaxOperationIn  int64  `db:"max_operation_in"`
	DailyMaxIn      int64  `db:"daily_max_in"`
	MonthlyMaxIn    int64  `db:"monthly_max_in"`
}

// AccountLimitsQ is a helper struct to aid in configuring queries that loads
// slices of AccountLimits structs.
type AccountLimitsQ struct {
	Err    error
	parent *Q
	sql    sq.SelectBuilder
}

type Commission struct {
	TotalOrderID
	KeyHash    string `db:"key_hash"`
	KeyValue   string `db:"key_value"`
	FlatFee    int64  `db:"flat_fee"`
	PercentFee int64  `db:"percent_fee"`
	weight     int
}

type AuditLog struct {
	Id        int64     `db:"id"`
	Actor     string    `db:"actor"`      //public key of the actor, performing task
	Subject   string    `db:"subject"`    //subject to change
	Action    string    `db:"action"`     //action performed on subject
	Meta      string    `db:"meta"`       //meta information about audit event
	CreatedAt time.Time `db:"created_at"` // time log was created
}

type Asset struct {
	Id          int64  `db:"id"`
	Type        int    `db:"type"`
	Code        string `db:"code"`
	Issuer      string `db:"issuer"`
	IsAnonymous bool   `db:"is_anonymous"`
}

// AssetQ is a helper struct to aid in configuring queries that loads
// slices of Assets.
type AssetQ struct {
	Err    error
	parent *Q
	sql    sq.SelectBuilder
}
