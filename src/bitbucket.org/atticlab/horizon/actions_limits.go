package horizon

import (
	"net/http"

	"bitbucket.org/atticlab/horizon/administration"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/render/hal"
	"bitbucket.org/atticlab/horizon/render/problem"
	"bitbucket.org/atticlab/horizon/resource"
	"bitbucket.org/atticlab/horizon/txsub"
)

type LimitsSetAction struct {
	Action
	Limits   history.AccountLimits
	Result   txsub.Result
	Resource resource.AccountLimits
}

// JSON format action handler
func (action *LimitsSetAction) JSON() {
	action.Do(
		action.loadLimits,
		action.updateLimits,
		action.loadResource,

		func() {
			hal.Render(action.W, action.Resource)
		})
}

func (action *LimitsSetAction) loadLimits() {
	action.ValidateBodyType()
	action.Limits.Account = action.GetString("account")
	action.Limits.AssetCode = action.GetString("asset_code")
	action.Limits.MaxOperation = action.GetInt64("max_operation")
	action.Limits.DailyTurnover = action.GetInt64("daily_turnnover")
	action.Limits.MonthlyTurnover = action.GetInt64("monthly_turnover")
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
