package horizon

import (
	"bitbucket.org/atticlab/horizon/assets"
	"bitbucket.org/atticlab/horizon/db2"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/db2/history/details"
	"bitbucket.org/atticlab/horizon/log"
	"bitbucket.org/atticlab/horizon/render/hal"
	"bitbucket.org/atticlab/horizon/resource"
)

// CommissionIndexAction returns a paged slice of commissions based upon the provided
// filters
type CommissionIndexAction struct {
	Action
	AccountFilter     string
	AccountTypeFilter *int32
	Asset             *details.Asset
	PagingParams      db2.PageQuery
	Records           []history.Commission
	Page              hal.Page
}

// JSON is a method for actions.JSON
func (action *CommissionIndexAction) JSON() {
	action.Do(action.loadParams, action.loadRecords, action.loadPage)
	action.Do(func() {
		hal.Render(action.W, action.Page)
	})
}

func (action *CommissionIndexAction) loadParams() {
	action.AccountFilter = action.GetString("account_id")
	action.AccountTypeFilter = action.GetInt32Pointer("account_type")
	action.PagingParams = action.GetPageQuery()
	if action.GetString("asset_type") != "" {
		xdrAsset := action.GetAsset("")
		action.Asset = new(details.Asset)
		*action.Asset = assets.ToBaseAsset(xdrAsset)
	}
}

func (action *CommissionIndexAction) loadRecords() {
	q := action.HistoryQ()
	comms := q.Commissions()

	switch {
	case action.AccountFilter != "":
		comms.ForAccount(action.AccountFilter)
	case action.AccountTypeFilter != nil:
		comms.ForAccountType(*action.AccountTypeFilter)
	case action.Asset != nil:
		comms.ForAsset(*action.Asset)
	}

	log.WithField("paging", action.PagingParams).Error("Selecting commission")
	action.Err = comms.Page(action.PagingParams).Select(&action.Records)
}

// loadPage populates action.Page
func (action *CommissionIndexAction) loadPage() {
	for _, record := range action.Records {
		var res resource.Commission
		action.Err = res.Populate(record)
		if action.Err != nil {
			return
		}
		action.Page.Add(res)
	}

	action.Page.BaseURL = action.BaseURL()
	action.Page.BasePath = action.Path()
	action.Page.Limit = action.PagingParams.Limit
	action.Page.Cursor = action.PagingParams.Cursor
	action.Page.Order = action.PagingParams.Order
	action.Page.PopulateLinks()
}
