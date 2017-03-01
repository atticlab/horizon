package horizon

import (
	"bitbucket.org/atticlab/horizon/db2/core"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/render/hal"
	"bitbucket.org/atticlab/horizon/resource"
)

// AccountStatisticsAction detailed income/outcome statistics for single account
type AccountStatisticsAction struct {
	Action
	Address       string
	AssetCode     string
	AssetIssuer   string
	HistoryRecord history.Account
	Statistics    []core.AccountStatistics
	Resource      resource.AccountStatistics
}

// JSON is a method for actions.JSON
func (action *AccountStatisticsAction) JSON() {
	action.Do(
		action.loadParams,
		action.loadRecord,
		action.loadResource,
		func() {
			hal.Render(action.W, action.Resource)
		},
	)
}

func (action *AccountStatisticsAction) loadParams() {
	action.Address = action.GetString("account_id")
	action.AssetCode = action.GetString("asset_code")
	action.AssetIssuer = action.GetString("asset_issuer")
}

func (action *AccountStatisticsAction) loadRecord() {
	action.Err = action.HistoryQ().AccountByAddress(&action.HistoryRecord, action.Address)
	if action.Err != nil {
		return
	}

	action.loadFromDB()

}

func (action *AccountStatisticsAction) loadFromDB() {
	q := action.CoreQ().Statistics()
	if action.Address != "" {
		q = q.ForAccount(action.Address)
	}

	if action.AssetIssuer != "" {
		q = q.ForAssetIssuer(action.AssetIssuer)
	}

	if action.AssetCode != "" {
		q = q.ForAssetCode(action.AssetCode)
	}

	action.Err = q.Select(&action.Statistics)
}

func (action *AccountStatisticsAction) loadResource() {
	action.Err = action.Resource.Populate(
		action.Ctx,
		action.Statistics,
		action.HistoryRecord,
	)
}
