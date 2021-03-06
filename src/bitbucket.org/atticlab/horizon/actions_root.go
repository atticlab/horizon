package horizon

import (
	"bitbucket.org/atticlab/horizon/render/hal"
	"bitbucket.org/atticlab/horizon/resource"
)

// RootAction provides a summary of the horizon instance and links to various
// useful endpoints
type RootAction struct {
	Action
}

// JSON renders the json response for RootAction
func (action *RootAction) JSON() {
	action.App.UpdateStellarCoreInfo()

	var res resource.Root
	res.Populate(
		action.Ctx,
		action.App.latestLedgerState.Horizon,
		action.App.latestLedgerState.Core,
		action.App.horizonVersion,
		action.App.coreVersion,
		action.App.networkPassphrase,
	)

	hal.Render(action.W, res)
}
