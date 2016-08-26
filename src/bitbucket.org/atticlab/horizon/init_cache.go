package horizon

import (
	"bitbucket.org/atticlab/horizon/cache"
	"time"
)

func initCache(app *App) {
	app.historyAccountCache = cache.NewHistoryAccountWithExp(app.HistoryQ(), time.Duration(2)*time.Minute, time.Duration(10)*time.Second)
}

func init() {
	appInit.Add("cache", initCache, "log", "horizon-db")
}
