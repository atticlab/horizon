package horizon

import (
	"bitbucket.org/atticlab/horizon/db2/core"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/txsub"
	"bitbucket.org/atticlab/horizon/txsub/results/db"
	"bitbucket.org/atticlab/horizon/txsub/sequence"
	"net/http"
)

func initSubmissionSystem(app *App) {
	cq := &core.Q{Repo: app.CoreRepo(nil)}
	hq := &history.Q{Repo: app.HorizonRepo(nil)}

	app.submitter = &txsub.System{
		Pending:         txsub.NewDefaultSubmissionList(),
		Submitter:       txsub.NewDefaultSubmitter(http.DefaultClient, app.config.StellarCoreURL, cq, hq, &app.config, app.SharedCache()),
		SubmissionQueue: sequence.NewManager(),
		Results: &results.DB{
			Core:    cq,
			History: hq,
		},
		Sequences:         cq.SequenceProvider(),
		NetworkPassphrase: app.networkPassphrase,
	}

	go func() {
		ticks := app.pump.Subscribe()

		for _ = range ticks {
			app.submitter.Tick(app.ctx)
		}
	}()

}

func init() {
	appInit.Add("txsub", initSubmissionSystem, "app-context", "log", "horizon-db", "core-db", "pump", "cache")
}
