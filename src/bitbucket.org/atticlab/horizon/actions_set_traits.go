package horizon

import (
	"bitbucket.org/atticlab/horizon/audit"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/render/hal"
	"bitbucket.org/atticlab/horizon/render/problem"
	"bitbucket.org/atticlab/horizon/resource"
	"database/sql"
)

// SetTraitsAction changes traits for specified account
type SetTraitsAction struct {
	Action
	Address  string
	BlockIn  *bool
	BlockOut *bool
	Result   error
	Resource resource.AccountTraits
}

// JSON format action handler
func (action *SetTraitsAction) JSON() {
	defer action.FinishAdminAction()
	action.Do(
		action.StartAdminAction,
		action.loadParams,
		action.updateTraits,
		action.loadResource,
		func() {
			hal.Render(action.W, action.Resource)
		})
}

func (action *SetTraitsAction) loadParams() {
	action.Address = action.GetAddress("account_id")
	action.BlockIn = action.GetOptionalBool("block_incoming_payments")
	action.BlockOut = action.GetOptionalBool("block_outcoming_payments")
}

func (action *SetTraitsAction) updateTraits() {
	action.adminAction.GetAuditInfo().Subject = audit.SubjectTraits
	// 1. Check if account exists
	var acc history.Account
	err := action.HistoryQ().AccountByAddress(&acc, action.Address)

	if err != nil {
		if err == sql.ErrNoRows {
			action.Err = &problem.NotFound
			return
		}
		action.Log.WithStack(err).WithError(err).Error("Failed to load account by address")
		action.Err = &problem.ServerError
		return
	}

	// 2. Try get traits for account
	var accTraits history.AccountTraits
	var isNew = false
	err = action.HistoryQ().GetAccountTraits(&accTraits, acc.ID)
	if err != nil {
		if err != sql.ErrNoRows {
			action.Err = &problem.ServerError
			action.Log.WithStack(err).WithError(err).Error("Failed to get account traits")
			return
		}
		isNew = true
		accTraits.ID = acc.ID
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

	action.adminAction.GetAuditInfo().Meta = accTraits
	// 4. Persist changes
	if isNew {
		err = action.HistoryQ().CreateAccountTraits(accTraits)
		action.adminAction.GetAuditInfo().ActionPerformed = audit.ActionPerformedInsert
	} else {
		err = action.HistoryQ().UpdateAccountTraits(accTraits)
		action.adminAction.GetAuditInfo().ActionPerformed = audit.ActionPerformedUpdate
	}

	if err != nil {
		action.Log.WithStack(err).WithError(err).Error("Failed to insert/update account traits")
		action.Err = &problem.ServerError
	}
}

func (action *SetTraitsAction) loadResource() {
	var traits history.AccountTraits
	err := action.HistoryQ().GetAccountTraitsByAddress(&traits, action.Address)

	if err == nil {
		action.Resource.Populate(action.Ctx, action.Address, traits)
		return
	}

	action.Log.WithStack(err).WithError(err).Error("Failed to GetAccountTraitsByAddress")
	action.Err = &problem.ServerError
}
