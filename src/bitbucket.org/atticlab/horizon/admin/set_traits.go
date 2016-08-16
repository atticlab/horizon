package admin

import (
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/render/problem"
	"database/sql"
)

type SetTraitsAction struct {
	AdminAction
	Address  string
	BlockIn  *bool
	BlockOut *bool

	accountTraits history.AccountTraits
	isNew         bool
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

	var err error
	action.accountTraits, err = action.HistoryQ().AccountTraitsQ().ForAccount(action.Address)
	if err != nil {
		if err != sql.ErrNoRows {
			action.Err = &problem.ServerError
			action.Log.WithStack(err).WithError(err).Error("Failed to get account traits")
			return
		}

		// account traits does not exists
		action.isNew = true
		action.accountTraits.BlockIncomingPayments = false
		action.accountTraits.BlockOutcomingPayments = false
	}

	//Set traits
	if action.BlockIn != nil {
		action.accountTraits.BlockIncomingPayments = *action.BlockIn
	}

	if action.BlockOut != nil {
		action.accountTraits.BlockOutcomingPayments = *action.BlockOut
	}

	if action.isNew {
		if action.toDelete() {
			action.Err = &problem.NotFound
			return
		}

		// Check if account exists
		var account history.Account
		err = action.HistoryQ().AccountByAddress(&account, action.Address)
		if err != nil {
			if err == sql.ErrNoRows {
				action.Err = &problem.NotFound
				return
			}

			action.Log.WithStack(err).WithError(err).Error("Failed to load account by address")
			action.Err = &problem.ServerError
			return
		}
		action.accountTraits.ID = account.ID
	}
}

func (action *SetTraitsAction) toDelete() bool {
	return !action.accountTraits.BlockIncomingPayments && !action.accountTraits.BlockOutcomingPayments
}

func (action *SetTraitsAction) Apply() {
	if action.Err != nil {
		return
	}

	var err error
	if action.isNew {
		err = action.HistoryQ().InsertAccountTraits(action.accountTraits)
	} else if action.toDelete() {
		err = action.HistoryQ().DeleteAccountTraits(action.accountTraits.ID)
	} else {
		err = action.HistoryQ().UpdateAccountTraits(action.accountTraits)
	}

	if err != nil {
		action.Log.WithStack(err).WithError(err).Error("Failed to insert/update account traits")
		action.Err = &problem.ServerError
		return
	}
}

func (action *SetTraitsAction) loadParams() {
	action.Address = action.GetAddress("account_id")
	action.BlockIn = action.GetOptionalBool("block_incoming_payments")
	action.BlockOut = action.GetOptionalBool("block_outcoming_payments")
}
