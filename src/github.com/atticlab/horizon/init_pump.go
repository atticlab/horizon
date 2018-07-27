package horizon

import (
	"time"

	"github.com/atticlab/horizon/pump"
	"github.com/atticlab/horizon/pump/db"
)

func initPump(app *App) {
	var trigger <-chan struct{}

	if app.config.Autopump {
		trigger = pump.Tick(1 * time.Second)
	} else {
		trigger = db.NewLedgerClosePump(app.ctx, app.HistoryQ())
	}

	app.pump = pump.NewPump(trigger)
}

func init() {
	appInit.Add("pump", initPump, "app-context", "log", "horizon-db", "core-db")
}
