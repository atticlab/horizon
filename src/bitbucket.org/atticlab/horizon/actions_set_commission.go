package horizon

import (
	"bitbucket.org/atticlab/horizon/assets"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/log"
	"bitbucket.org/atticlab/horizon/render/problem"
	"net/http"
)

// Inserts new Commission if CommissionId is 0, otherwise - tries to update
type SetCommissionAction struct {
	Action
	CommissionKey history.CommissionKey
	FlatFee       int64
	PercentFee    int64
	CommissionId  int64
}

// JSON format action handler
func (action *SetCommissionAction) JSON() {
	action.Do(
		action.loadCommission,
		action.updateCommission)
}

func (action *SetCommissionAction) loadCommission() {
	action.ValidateBodyType()
	action.CommissionKey.From = action.GetString("from")
	action.CommissionKey.To = action.GetString("to")
	action.CommissionKey.FromType = action.GetInt32Pointer("from_type")
	action.CommissionKey.ToType = action.GetInt32Pointer("to_type")
	if action.GetString("asset_type") != "" {
		xdrAsset := action.GetAsset("")
		if action.Err != nil {
			return
		}
		action.CommissionKey.Asset = assets.ToBaseAsset(xdrAsset)
	}
	action.FlatFee = action.GetInt64("flat_fee")
	action.PercentFee = action.GetInt64("percent_fee")
	action.CommissionId = action.GetInt64("id")
	log.WithField("key", action.CommissionKey).Debug("got params")
}

func (action *SetCommissionAction) updateCommission() {
	if action.Err != nil {
		return
	}
	log.Debug("Updating commission")
	commission, err := history.NewCommission(action.CommissionKey, action.FlatFee, action.PercentFee)
	if err != nil {
		log.WithStack(err).WithError(err).Error("Failed to create new commission")
		action.Err = &problem.P{
			Type:   "invalid_commission_key",
			Title:  "Invalid commission key",
			Status: http.StatusBadRequest,
			Detail: "Horizon could not create commission, because commission key is invalid.",
		}
		return
	}
	commission.Id = action.CommissionId

	if commission.Id != 0 {
		log.WithField("commission", commission).Debug("Trying to update commission")
		var updated bool
		updated, err = action.HistoryQ().UpdateCommission(commission)
		if err == nil && !updated {
			action.Err = &problem.P{
				Type:   "not_found",
				Title:  "Commission with such id not found",
				Status: http.StatusNotFound,
				Detail: "Horizon could not update commission, because commission with such id was not found.",
			}
			return
		}
	} else {
		log.WithField("commission", commission).Debug("Trying to insert commission")
		err = action.HistoryQ().InsertCommission(commission)
	}

	if err != nil {
		action.Err = &problem.P{
			Type:   "internal_error",
			Title:  "Internal Server Error",
			Status: http.StatusInternalServerError,
		}
	}
}
