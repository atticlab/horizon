package horizon

import (
	"net/http"

	"bitbucket.org/atticlab/horizon/administration"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/render/hal"
	"bitbucket.org/atticlab/horizon/render/problem"
	"bitbucket.org/atticlab/horizon/render/sse"
	"bitbucket.org/atticlab/horizon/resource"
	"bitbucket.org/atticlab/horizon/txsub"
)

// // LimitsAction detailed income/outcome limits for all accounts
// type LimitsAction struct {
// 	Action
// 	Limits   map[string][]history.AccountLimits
// 	Resource resource.AccountLimits
// }

// // JSON is a method for actions.JSON
// func (action *LimitsAction) JSON() {
// 	action.Do(
// 		action.loadParams,
// 		action.loadRecord,
// 		action.loadResource,
// 		func() {
// 			hal.Render(action.W, action.Resource)
// 		},
// 	)
// }

// // SSE is a method for actions.SSE
// func (action *LimitsAction) SSE(stream sse.Stream) {
// 	// TODO: check
// 	action.Do(
// 		action.loadParams,
// 		action.loadRecord,
// 		action.loadResource,
// 		func() {
// 			stream.Send(sse.Event{Data: action.Resource})
// 		},
// 	)
// }

// func (action *LimitsAction) loadParams() {
// 	action.Address = action.GetString("account_id")
// }

// func (action *LimitsAction) loadRecord() {
// 	action.Err = action.HistoryQ().GetLimitsByAccount(&action.AccountLimits, action.Address)
// 	if action.Err != nil {
// 		return
// 	}
// }

// func (action *LimitsAction) loadResource() {
// 	action.Err = action.Resource.Populate(
// 		action.Ctx,
// 		action.Address,
// 		action.AccountLimits,
// 	)
// }

// AccountLimitsAction detailed income/outcome limits for single account
type AccountLimitsAction struct {
	Action
	Address       string
	AccountLimits []history.AccountLimits
	Resource      resource.AccountLimits
}

// JSON is a method for actions.JSON
func (action *AccountLimitsAction) JSON() {
	action.Do(
		action.loadParams,
		action.loadRecord,
		action.loadResource,
		func() {
			hal.Render(action.W, action.Resource)
		},
	)
}

// SSE is a method for actions.SSE
func (action *AccountLimitsAction) SSE(stream sse.Stream) {
	// TODO: check
	action.Do(
		action.loadParams,
		action.loadRecord,
		action.loadResource,
		func() {
			stream.Send(sse.Event{Data: action.Resource})
		},
	)
}

func (action *AccountLimitsAction) loadParams() {
	action.Address = action.GetString("account_id")
}

func (action *AccountLimitsAction) loadRecord() {
	action.Err = action.HistoryQ().GetLimitsByAccount(&action.AccountLimits, action.Address)
	if action.Err != nil {
		return
	}
}

func (action *AccountLimitsAction) loadResource() {
	action.Err = action.Resource.Populate(
		action.Ctx,
		action.Address,
		action.AccountLimits,
	)
}

type LimitsSetAction struct {
	AdminAction
	Limits   history.AccountLimits
	Result   txsub.Result
	Resource resource.AccountLimits
}

// JSON format action handler
func (action *LimitsSetAction) JSON() {
	action.Do(
		action.StartAdminAction,
		action.loadLimits,
		action.updateLimits,
		action.loadResource,
		action.FinishAdminAction,

		func() {
			hal.Render(action.W, action.Resource)
		})
}

func (action *LimitsSetAction) loadLimits() {

	action.ValidateBodyType()
	action.Limits.Account = action.GetString("account_id")
	action.Limits.AssetCode = action.GetString("asset_code")
	action.Limits.MaxOperationOut = action.GetInt64("max_operation_out")
	action.Limits.DailyMaxOut = action.GetInt64("daily_max_out")
	action.Limits.MonthlyMaxOut = action.GetInt64("monthly_max_out")
	action.Limits.MaxOperationIn = action.GetInt64("max_operation_in")
	action.Limits.DailyMaxIn = action.GetInt64("daily_max_in")
	action.Limits.MonthlyMaxIn = action.GetInt64("monthly_max_in")
}

func (action *LimitsSetAction) updateLimits() {
	result := (*action.App.AccountManager()).SetLimits(action.Limits)

	switch err := result.(type) {
	case administration.AccountNotFoundError:
		action.Err = &problem.P{
			Type:   "account_not_found",
			Title:  "Account not found",
			Status: http.StatusNotFound,
			Detail: "Horizon could not set limits for account, because it wasn't found.",
			Extras: map[string]interface{}{
				"account_id": action.Limits.Account,
			},
		}
	default:
		action.Err = err
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
