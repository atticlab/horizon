package horizon

import (
	"bitbucket.org/atticlab/horizon/log"
	"bitbucket.org/atticlab/horizon/redis"
)

func initRedis(app *App) {
	err := redis.Init(app.config.RedisURL)
	if err != nil {
		log.WithField("service", "redis").WithError(err).Panic("Failed to initialize")
	}
	app.redis = redis.GetPool()
}

func init() {
	appInit.Add("redis", initRedis, "app-context", "log")
}
