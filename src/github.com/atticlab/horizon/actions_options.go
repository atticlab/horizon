package horizon

import (
	"github.com/atticlab/horizon/db2/history"
	"github.com/atticlab/horizon/render/hal"
	"github.com/atticlab/horizon/resource/options"
	"github.com/atticlab/horizon/txsub/transactions"
	"github.com/atticlab/horizon/render/problem"
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
