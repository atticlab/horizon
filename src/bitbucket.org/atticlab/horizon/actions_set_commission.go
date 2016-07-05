package horizon

import (
	"bitbucket.org/atticlab/horizon/assets"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/log"
	"bitbucket.org/atticlab/horizon/render/hal"
	"bitbucket.org/atticlab/horizon/render/problem"
	"github.com/go-errors/errors"
	"net/http"
	"bitbucket.org/atticlab/horizon/audit"
)

// Inserts new Commission if CommissionId is 0, otherwise - tries to update
type SetCommissionAction struct {
	AdminAction
	CommissionKey history.CommissionKey
	FlatFee       int64
	PercentFee    int64
	CommissionId  int64
	Delete        bool
}

// JSON format action handler
func (action *SetCommissionAction) JSON() {
	action.Do(
		action.StartAdminAction,
		action.loadCommission,
		action.updateCommission,
		action.FinishAdminAction,
		func() {
			if action.Err == nil {
				hal.Render(action.W, problem.P{
					Status: 200,
				})
			}
		})
}

func (action *SetCommissionAction) getOptionalAccountID(name string) string {
	if action.Err != nil {
		return ""
	}
	accountID := action.GetOptionalAccountID(name)
	if accountID == nil || action.Err != nil {
		return ""
	}
	return accountID.Address()
}

func (action *SetCommissionAction) getOptionalRawAccountType(name string) *int32 {
	if action.Err != nil {
		return nil
	}
	accountType := action.GetOptionalAccountType(name)
	if accountType == nil {
		return nil
	}
	rawAccountType := int32(*accountType)
	return &rawAccountType
}

func (action *SetCommissionAction) loadCommission() {
	if action.Err != nil {
		return
	}
	action.ValidateBodyType()
	action.CommissionKey.From = action.getOptionalAccountID("from")
	action.CommissionKey.To = action.getOptionalAccountID("to")
	action.CommissionKey.FromType = action.getOptionalRawAccountType("from_type")
	action.CommissionKey.ToType = action.getOptionalRawAccountType("to_type")
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
	action.Info.Subject = audit.SubjectCommission
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

	action.Info.Meta = commission
	if commission.Id == 0 {
		if action.Delete {
			action.Err = &problem.NotFound
			return
		}
		log.WithField("commission", commission).Debug("Trying to insert commission")
		action.Info.ActionPerformed = audit.ActionPerformedInsert
		err = action.HistoryQ().InsertCommission(commission)
		if err != nil {
			log.WithField("commission", commission).WithError(err).Error("Failed to insert new commission")
			action.Err = &problem.ServerError
		}
		return
	}

	var updated bool
	if action.Delete {
		action.Info.ActionPerformed = audit.ActionPerformedDelete
		log.WithField("commissionid", commission.Id).Debug("Trying to delete commission")
		updated, err = action.HistoryQ().DeleteCommission(commission.Id)
	} else {
		action.Info.ActionPerformed = audit.ActionPerformedUpdate
		log.WithField("commission", commission).Debug("Trying to update commission")
		updated, err = action.HistoryQ().UpdateCommission(commission)
	}

	if err != nil {
		log.WithField("commission", commission).WithField("delete", action.Delete).WithError(err).Error("Failed to update/delete commission")
		action.Err = &problem.ServerError
		return
	}

	if !updated {
		action.Err = &problem.NotFound
	}
}
