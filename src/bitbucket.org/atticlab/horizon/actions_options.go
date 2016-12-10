package horizon

import (
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/render/hal"
	"bitbucket.org/atticlab/horizon/resource/options"
	"bitbucket.org/atticlab/horizon/txsub/transactions"
	"bitbucket.org/atticlab/horizon/render/problem"
)

// OptionsAction renders options.
type OptionsAction struct {
	Action
	Response options.Options
}

// JSON is a method for actions.JSON
func (action *OptionsAction) JSON() {
	action.Do(
		action.loadRecord,
		func() {
			hal.Render(action.W, action.Response)
		},
	)
}

func (action *OptionsAction) loadRecord() {
	rawMaxReversalDuration, err := action.HistoryQ().OptionsByName(history.OPTIONS_MAX_REVERSAL_DURATION)
	if err != nil {
		action.Log.WithError(err).Error("Failed to get max reversal duration")
		action.Err = &problem.ServerError
		return
	}

	var maxReversalDuration history.MaxReversalDuration
	if rawMaxReversalDuration != nil {
		maxReversalDuration = history.MaxReversalDuration(*rawMaxReversalDuration)
	} else {
		maxReversalDuration = *history.NewMaxReversalDuration()
		maxReversalDuration.SetMaxDuration(transactions.MAX_REVERSE_TIME)
	}

	action.Response.MaxReversalDuration.Populate(maxReversalDuration)
}
