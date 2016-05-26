package horizon

import (
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/administration"
)

func initAdministrationSystem(app *App) {
	hq := &history.Q{Repo: app.HorizonRepo(nil)}
    admin := administration.NewAccountManager(hq,  &app.config)
    app.accountManager = &admin
}

func init() {
	appInit.Add("administration", initAdministrationSystem, "app-context", "log", "horizon-db")
}