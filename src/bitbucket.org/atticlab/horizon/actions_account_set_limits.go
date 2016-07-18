package horizon

import (
	"bitbucket.org/atticlab/horizon/audit"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/log"
	"bitbucket.org/atticlab/horizon/render/hal"
	"bitbucket.org/atticlab/horizon/render/problem"
	"bitbucket.org/atticlab/horizon/resource"
	"database/sql"
)

type LimitsSetAction struct {
	Action
	Limits   history.AccountLimits
	Resource resource.AccountLimits
}

// JSON format action handler
func (action *LimitsSetAction) JSON() {
	defer action.FinishAdminAction()
	action.Do(
		action.StartAdminAction,
		action.loadLimits,
		action.updateLimits,
		action.loadResource,
		func() {
			hal.Render(action.W, action.Resource)
		})
}

func (action *LimitsSetAction) loadLimits() {
	action.ValidateBodyType()
	action.Limits.Account = action.GetAddress("account_id")
	action.Limits.AssetCode = action.GetString("asset_code")
	action.Limits.MaxOperationOut = action.GetInt64("max_operation_out")
	action.Limits.DailyMaxOut = action.GetInt64("daily_max_out")
	action.Limits.MonthlyMaxOut = action.GetInt64("monthly_max_out")
	action.Limits.MaxOperationIn = action.GetInt64("max_operation_in")
	action.Limits.DailyMaxIn = action.GetInt64("daily_max_in")
	action.Limits.MonthlyMaxIn = action.GetInt64("monthly_max_in")
	action.adminAction.GetAuditInfo().Subject = audit.SubjectAccountLimits
}

func (action *LimitsSetAction) updateLimits() {
	// 1. Check if account exists
	var acc history.Account
	err := action.HistoryQ().AccountByAddress(&acc, action.Limits.Account)

	if err != nil {
		if err == sql.ErrNoRows {
			action.Err = &problem.NotFound
			return
		}
		log.WithStack(err).WithError(err).Error("Failed to load account by address")
		action.Err = &problem.ServerError
		return
	}

	// 2. Try get limits for account
	var isNewEntry bool
	var accLimits history.AccountLimits
	err = action.HistoryQ().GetAccountLimits(&accLimits, action.Limits.Account, action.Limits.AssetCode)
	if err != nil {
		if err != sql.ErrNoRows {
			log.WithStack(err).WithError(err).Error("Failed to get account limits")
			action.Err = &problem.ServerError
			return
		}
		isNewEntry = true
	}
	// 3. Validate and set limits
	accLimits = action.Limits

	action.adminAction.GetAuditInfo().Meta = accLimits
	// 4. Persist changes
	if isNewEntry {
		action.adminAction.GetAuditInfo().ActionPerformed = audit.ActionPerformedInsert
		err = action.HistoryQ().CreateAccountLimits(accLimits)
	} else {
		action.adminAction.GetAuditInfo().ActionPerformed = audit.ActionPerformedUpdate
		err = action.HistoryQ().UpdateAccountLimits(accLimits)
	}

	if err != nil {
		log.WithStack(err).WithError(err).Error("Failed to insert/update account limits")
		action.Err = &problem.ServerError
	}
}

func (action *LimitsSetAction) loadResource() {
	var limits []history.AccountLimits
	action.Err = (*action.HistoryQ()).GetLimitsByAccount(&limits, action.Limits.Account)

	if action.Err == nil {
		action.Resource.Populate(action.Ctx, action.Limits.Account, limits)
		return
	}
}
