package admin

import (
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/render/problem"
	"database/sql"
)

type SetTraitsAction struct {
	AdminAction
	Account  history.Account
	Address  string
	BlockIn  *bool
	BlockOut *bool
}

func NewSetTraitsAction(adminAction AdminAction) *SetTraitsAction {
	return &SetTraitsAction{
		AdminAction: adminAction,
	}
}

func (action *SetTraitsAction) Validate() {
	action.loadParams()
	if action.Err != nil {
		return
	}
	// 1. Check if account exists
	err := action.HistoryQ().AccountByAddress(&action.Account, action.Address)

	if err != nil {
		if err == sql.ErrNoRows {
			action.Err = &problem.NotFound
			return
		}
		action.Log.WithStack(err).WithError(err).Error("Failed to load account by address")
		action.Err = &problem.ServerError
		return
	}
}

func (action *SetTraitsAction) Apply() {
	if action.Err != nil {
		return
	}

	// 2. Try get traits for account
	var accTraits history.AccountTraits
	var isNew = false
	err := action.HistoryQ().GetAccountTraits(&accTraits, action.Account.ID)
	if err != nil {
		if err != sql.ErrNoRows {
			action.Err = &problem.ServerError
			action.Log.WithStack(err).WithError(err).Error("Failed to get account traits")
			return
		}
		isNew = true
		accTraits.ID = action.Account.ID
		accTraits.BlockIncomingPayments = false
		accTraits.BlockOutcomingPayments = false
	}

	// 3. Set traits
	if action.BlockIn != nil {
		accTraits.BlockIncomingPayments = *action.BlockIn
	}

	if action.BlockOut != nil {
		accTraits.BlockOutcomingPayments = *action.BlockOut
	}

	// 4. Persist changes
	if isNew {
		err = action.HistoryQ().CreateAccountTraits(accTraits)
	} else {
		err = action.HistoryQ().UpdateAccountTraits(accTraits)
	}

	if err != nil {
		action.Log.WithStack(err).WithError(err).Error("Failed to insert/update account traits")
		action.Err = &problem.ServerError
	}
}

func (action *SetTraitsAction) loadParams() {
	action.Address = action.GetAddress("account_id")
	action.BlockIn = action.GetOptionalBool("block_incoming_payments")
	action.BlockOut = action.GetOptionalBool("block_outcoming_payments")
}
