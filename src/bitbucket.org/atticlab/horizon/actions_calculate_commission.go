package horizon

import (
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/commissions"
	"bitbucket.org/atticlab/horizon/log"
	"bitbucket.org/atticlab/horizon/render/hal"
	"bitbucket.org/atticlab/horizon/render/problem"
	"bitbucket.org/atticlab/horizon/resource/operations"
	"database/sql"
)

type CalculateCommissionAction struct {
	Action
	source      xdr.AccountId
	destination xdr.AccountId
	amount      xdr.Int64
	asset       xdr.Asset
	Resource    operations.Fee
}

// JSON format action handler
func (action *CalculateCommissionAction) JSON() {
	action.Do(
		action.loadParams,
		action.calculate,
		func() {
			hal.Render(action.W, action.Resource)
		})
}

func (action *CalculateCommissionAction) loadParams() {
	action.source = action.GetAccountID("from")
	action.destination = action.GetAccountID("to")
	action.asset = action.GetAsset("")
	action.amount = action.GetPositiveAmount("amount")
}

func (action *CalculateCommissionAction) calculate() {
	if action.Err != nil {
		return
	}
	log := log.WithFields(log.F{
		"from":   action.source.Address(),
		"to":     action.destination.Address(),
		"amount": action.amount,
		"asset":  action.asset,
	})
	cm := commissions.New(action.App.SharedCache(), action.HistoryQ())
	fee, err := cm.CalculateCommission(action.source, action.destination, action.amount, action.asset)
	if err != nil {
		if err == sql.ErrNoRows {
			action.Err = &problem.NotFound
			return
		}
		log.WithError(err).Error("Failed to count fee")
		action.Err = &problem.ServerError
		return
	}
	action.Resource.Populate(*fee)
}
