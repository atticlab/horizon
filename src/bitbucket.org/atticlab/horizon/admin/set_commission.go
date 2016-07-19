package admin

import (
	"bitbucket.org/atticlab/horizon/assets"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/render/problem"
	"errors"
)

type SetCommissionAction struct {
	AdminAction
	CommissionId  int64
	CommissionKey history.CommissionKey
	FlatFee       int64
	PercentFee    int64
	Delete        bool
	commission    *history.Commission
}

func NewSetCommissionAction(adminAction AdminAction) *SetCommissionAction {
	return &SetCommissionAction{
		AdminAction: adminAction,
	}
}

func (action *SetCommissionAction) Validate() {
	action.loadParams()
	if action.HasError() {
		return
	}

	var err error
	action.commission, err = history.NewCommission(action.CommissionKey, action.FlatFee, action.PercentFee)
	if err != nil {
		action.Log.WithStack(err).WithError(err).Error("Failed to create new commission")
		action.Err = errors.New("invalid commission_key")
		return
	}

	action.commission.Id = action.CommissionId

	if action.commission.Id == 0 && action.Delete {
		action.Err = &problem.NotFound
		return
	}

	if action.commission.Id != 0 {
		stored, err := action.HistoryQ().CommissionById(action.CommissionId)
		if err != nil {
			action.Log.WithStack(err).WithError(err).Error("Failed to get commission by id")
			action.Err = &problem.ServerError
			return
		}

		if stored == nil {
			action.Err = &problem.NotFound
			return
		}
	}
}

func (action *SetCommissionAction) Apply() {
	if action.Err != nil {
		return
	}
	action.Log.WithField("commission", action.commission).Debug("Updating commission")
	var err error
	action.commission.Id = action.CommissionId

	if action.commission.Id == 0 {
		action.Log.WithField("commission", action.commission).Debug("Trying to insert commission")
		err = action.HistoryQ().InsertCommission(action.commission)
		if err != nil {
			action.Log.WithField("commission", action.commission).WithError(err).Error("Failed to insert new commission")
			action.Err = &problem.ServerError
		}
		return
	}

	var updated bool
	if action.Delete {
		action.Log.WithField("commissionid", action.commission.Id).Debug("Trying to delete commission")
		updated, err = action.HistoryQ().DeleteCommission(action.commission.Id)
	} else {
		action.Log.WithField("commission", action.commission).Debug("Trying to update commission")
		updated, err = action.HistoryQ().UpdateCommission(action.commission)
	}

	if err != nil {
		action.Log.WithField("commission", action.commission).WithField("delete", action.Delete).WithError(err).Error("Failed to update/delete commission")
		action.Err = &problem.ServerError
		return
	}

	if !updated {
		action.Err = &problem.NotFound
	}
}

func (action *SetCommissionAction) loadParams() {
	action.CommissionKey.From = action.GetOptionalAddress("from")
	action.CommissionKey.To = action.GetOptionalAddress("to")
	action.CommissionKey.FromType = action.GetOptionalRawAccountType("from_type")
	action.CommissionKey.ToType = action.GetOptionalRawAccountType("to_type")
	if action.GetString("asset_type") != "" {
		xdrAsset := action.GetAsset("")
		if action.Err != nil {
			return
		}
		action.CommissionKey.Asset = assets.ToBaseAsset(xdrAsset)
	}
	action.FlatFee = action.GetInt64("flat_fee")
	if action.FlatFee < 0 {
		action.SetInvalidField("flat_fee", errors.New("flat_fee can not be negative"))
		return
	}
	action.PercentFee = action.GetInt64("percent_fee")
	if action.PercentFee < 0 {
		action.SetInvalidField("percent_fee", errors.New("percent_fee can not be negative"))
		return
	}
	action.CommissionId = action.GetInt64("id")
	action.Delete = action.GetBool("delete")
}
