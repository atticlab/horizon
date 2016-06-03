package horizon

import (
	"net/http"

	"bitbucket.org/atticlab/horizon/administration"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/render/hal"
	"bitbucket.org/atticlab/horizon/render/problem"
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

// SetTraitsAction changes traits for specified account
type SetTraitsAction struct {
	Action
	Address  string
	Traits   map[string]string
	Result   error
	Resource resource.AccountTraits
}

// JSON format action handler
func (action *SetTraitsAction) JSON() {
	action.Do(
		action.requireAdminSignature,
		action.loadParams,
		action.updateTraits,
		action.loadResource,

		func() {
			hal.Render(action.W, action.Resource)
		})
}

func (action *SetTraitsAction) loadParams() {
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

// type AccountNotFoundError struct {
//     Address string
// }

// func (err AccountNotFoundError) Error() string {
//     return fmt.Sprintf("Account with address %s wasn't found.", err.Address)
// }

// // InvalidFieldsError contains array if errors, corresponding to request fields
// type InvalidFieldsError struct {
