package history

import (
	sq "github.com/lann/squirrel"
)

// GetAccountTraitsByAddress returns traits for specified account
func (q *Q) GetAccountTraitsByAddress(dest interface{}, accountID string) error {
    var acc Account
    err := q.AccountByAddress(&acc, accountID)
    if err != nil {
        return err
    }
    
    return q.GetAccountTraits(dest, acc.ID)
}

// GetAccountTraits returns traits for specified account
func (q *Q) GetAccountTraits(dest interface{}, id int64) error {
    sql := selectAccountTraits.Limit(1).Where("at.id = ?", id)
    return q.Get(dest, sql)
}

// CreateAccountTraits inserts new account_traits row
func (q *Q) CreateAccountTraits(traits AccountTraits) error {
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

var selectAccountTraits = sq.Select("at.*").From("account_traits at")
var createAccountTraits = sq.Insert("account_traits").Columns(
	"id",
	"block_incoming_payments",
    "block_outcoming_payments",
)
var updateAccountTraits = sq.Update("account_traits")
