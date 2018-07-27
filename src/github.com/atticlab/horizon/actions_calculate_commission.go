package horizon

import (
	"github.com/atticlab/go-smart-base/xdr"
	"github.com/atticlab/horizon/commissions"
	"github.com/atticlab/horizon/db2/history/details"
	"github.com/atticlab/horizon/log"
	"github.com/atticlab/horizon/render/hal"
	"github.com/atticlab/horizon/render/problem"
	"database/sql"
)

type CalculateCommissionAction struct {
	Action
	source      xdr.AccountId
	destination xdr.AccountId
	amount      xdr.Int64
	asset       xdr.Asset
	Resource    details.Fee
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
