package horizon

import (
	"bitbucket.org/atticlab/horizon/administration"
	"bitbucket.org/atticlab/horizon/resource"
	"bitbucket.org/atticlab/horizon/render/hal"
	"bitbucket.org/atticlab/horizon/render/problem"
	"net/http"
	"bitbucket.org/atticlab/horizon/db2/history"
)

//TODO CHECK!!
// SetTraitsAction changes traits for specified account
type SetTraitsAction struct {
	AdminAction
	Address  string
	Traits   map[string]string
	Result   error
	Resource resource.AccountTraits
}

// JSON format action handler
func (action *SetTraitsAction) JSON() {
	action.Do(
		action.StartAdminAction,
		action.loadParams,
		action.updateTraits,
		action.loadResource,
		action.FinishAdminAction,

		func() {
			hal.Render(action.W, action.Resource)
		})
}

func (action *SetTraitsAction) loadParams() {
	if action.Err != nil {
		return
	}
	action.Address = action.GetAddress("account_id")
	action.Traits = make(map[string]string)

	// TODO: move all validation logic here
	blockIncoming := action.GetString("block_incoming_payments")
	if len(blockIncoming) > 0 {
		action.Traits["block_incoming_payments"] = blockIncoming
	}

	blockOutcoming := action.GetString("block_outcoming_payments")
	if len(blockOutcoming) > 0 {
		action.Traits["block_outcoming_payments"] = blockOutcoming
	}
}

func (action *SetTraitsAction) updateTraits() {
	if action.Err == nil {
		return
	}
	result := (*action.App.AccountManager()).SetTraits(action.Address, action.Traits)

	switch err := result.(type) {
	case administration.AccountNotFoundError:
		println("Sup")
		action.Err = &problem.P{
			Type:   "account_not_found",
			Title:  "Account not found",
			Status: http.StatusNotFound,
			Detail: "Horizon could not set traits for account, because it wasn't found.",
			Extras: map[string]interface{}{
				"account_id": action.Address,
			},
		}
	case administration.InvalidFieldsError:
		println("Soap")
		action.Err = &problem.P{
			Type:   "malformed_request",
			Title:  "Malformed request",
			Status: http.StatusBadRequest,
			Detail: "Request contains some invalid fields. See 'Extras' for details.",
			Extras: extractErrors(err.Errors),
		}
	default:
		action.Err = err
	}
}

func (action *SetTraitsAction) loadResource() {
	if action.Err != nil {
		return
	}
	var traits history.AccountTraits
	action.Err = (*action.HistoryQ()).GetAccountTraitsByAddress(&traits, action.Address)

	if action.Err == nil {
		action.Resource.Populate(action.Ctx, action.Address, traits)
		return
	}
}

func extractErrors(errors map[string]error) map[string]interface{} {
	result := make(map[string]interface{})
	for key, err := range errors {
		result[key] = err.Error()
	}

	return result
}