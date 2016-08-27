package horizon

import (
	"bitbucket.org/atticlab/horizon/cache"
	"bitbucket.org/atticlab/horizon/db2/history"
	"time"
)

func initCacheFromApp(app *App) {
	app.historyAccountCache = initCache(app.HistoryQ())
}

func initCache(history *history.Q) *cache.HistoryAccount {
	return cache.NewHistoryAccountWithExp(history, time.Duration(2)*time.Minute, time.Duration(10)*time.Second)
}

func init() {
	appInit.Add("cache", initCacheFromApp, "log", "horizon-db")
}
