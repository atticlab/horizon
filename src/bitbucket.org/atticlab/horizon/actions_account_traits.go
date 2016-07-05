package horizon

import (

	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/render/hal"
	"bitbucket.org/atticlab/horizon/render/sse"
	"bitbucket.org/atticlab/horizon/resource"
)

// AccountTraitsAction detailed income/outcome statistics for single account
type AccountTraitsAction struct {
	Action
	Address       string
	AccountTraits history.AccountTraits
	Resource      resource.AccountTraits
}

// JSON is a method for actions.JSON
func (action *AccountTraitsAction) JSON() {
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
func (action *AccountTraitsAction) SSE(stream sse.Stream) {
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

func (action *AccountTraitsAction) loadParams() {
	action.Address = action.GetString("account_id")
}

func (action *AccountTraitsAction) loadRecord() {
	action.Err = action.HistoryQ().GetAccountTraitsByAddress(&action.AccountTraits, action.Address)
	if action.Err != nil {
		return
	}
}

func (action *AccountTraitsAction) loadResource() {
	action.Err = action.Resource.Populate(
		action.Ctx,
		action.Address,
		action.AccountTraits,
	)
}
