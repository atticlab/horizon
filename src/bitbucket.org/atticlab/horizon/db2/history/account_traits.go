package history

import (
	"bitbucket.org/atticlab/horizon/db2"
	sq "github.com/lann/squirrel"
)

// AccountTraits is a row of data from the `account_traits` table
type AccountTraits struct {
	TotalOrderID
	AccountAddress         string `db:"address"`
	BlockIncomingPayments  bool   `db:"block_incoming_payments"`
	BlockOutcomingPayments bool   `db:"block_outcoming_payments"`
}

type AccountTraitsQInterface interface {
	// Selects account's traits by account address
	ForAccount(aid string) (traits AccountTraits, err error)
	// Selects account's traits by id
	ByID(id int64) (traits AccountTraits, err error)
	// And page filters to query
	Page(page db2.PageQuery) AccountTraitsQInterface
	// Selects account's traits
	Select(dest interface{}) error
}

// AccountTraitsQ is a helper struct to aid in configuring queries that loads
// slices of AccountTraits structs.
type AccountTraitsQ struct {
	Err    error
	parent *Q
	sql    sq.SelectBuilder
}

// AccountTraitsQ provides a helper to filter the operations table with pre-defined
// filters.  See `AccountTraitsQ` for the available filters.
func (q *Q) AccountTraitsQ() AccountTraitsQInterface {
	return &AccountTraitsQ{
		parent: q,
		sql:    selectAccountTraits,
	}
}

// Selects AccountTraits by Account.
func (q *AccountTraitsQ) ForAccount(aid string) (traits AccountTraits, err error) {
	if q.Err != nil {
		return traits, q.Err
	}

	q.sql = q.sql.Limit(1).Where("ha.address = ?", aid)
	q.Err = q.parent.Get(&traits, q.sql)
	return traits, q.Err
}

// Selects AccountTraits by ID
func (q *AccountTraitsQ) ByID(id int64) (traits AccountTraits, err error) {
	if q.Err != nil {
		return traits, q.Err
	}

	q.sql = q.sql.Limit(1).Where("at.id = ?", id)
	q.Err = q.parent.Get(&traits, q.sql)
	return traits, q.Err
}

// Page specifies the paging constraints for the query being built by `q`.
func (q *AccountTraitsQ) Page(page db2.PageQuery) AccountTraitsQInterface {
	if q.Err != nil {
		return q
	}

	q.sql, q.Err = page.ApplyTo(q.sql, "at.id")
	return q
}

// Select loads the results of the query specified by `q` into `dest`.
func (q *AccountTraitsQ) Select(dest interface{}) error {
	if q.Err != nil {
		return q.Err
	}

	q.Err = q.parent.Select(dest, q.sql)
	return q.Err
}

// InsertAccountTraits inserts new account_traits row
func (q *Q) InsertAccountTraits(traits AccountTraits) error {
	sql := createAccountTraits.Values(traits.ID, traits.BlockIncomingPayments, traits.BlockOutcomingPayments)
	_, err := q.Exec(sql)

	return err
}

// UpdateAccountTraits updates account_traits row
func (q *Q) UpdateAccountTraits(traits AccountTraits) error {
	sql := updateAccountTraits.Set("block_incoming_payments", traits.BlockIncomingPayments)
	sql = sql.Set("block_outcoming_payments", traits.BlockOutcomingPayments)
	sql = sql.Where("id = ?", traits.ID)

	_, err := q.Exec(sql)

	return err
}

func (q *Q) DeleteAccountTraits(id int64) error {
	sql := deleteAccountTraits.Where("at.id = ?", id)
	_, err := q.Exec(sql)
	return err
}

var selectAccountTraits = sq.Select("at.*, ha.address").From("account_traits at").Join("history_accounts ha ON at.id = ha.id")
var createAccountTraits = sq.Insert("account_traits").Columns(
	"id",
	"block_incoming_payments",
	"block_outcoming_payments",
)
var updateAccountTraits = sq.Update("account_traits")
var deleteAccountTraits = sq.Delete("account_traits at")
