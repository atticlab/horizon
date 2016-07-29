package horizon

import (
	"github.com/garyburd/redigo/redis"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	"bitbucket.org/atticlab/horizon/test"
)

func TestRedis(t *testing.T) {

	Convey("app.redis gets set when RedisURL is set", t, func() {
		c := test.NewTestConfig()
		c.RedisURL = "redis://127.0.0.1:6379/"
		app, _ := NewApp(c)
		defer app.Close()
		So(app.redis, ShouldNotBeNil)
	})

	Convey("app.redis is nil when no RedisURL is set", t, func() {
		c := test.NewTestConfig()
		c.RedisURL = ""
		app, _ := NewApp(c)
		defer app.Close()
		So(app.redis, ShouldBeNil)
	})

	Convey("app.redis can successfully connect to redis", t, func() {
		conf := test.NewTestConfig()
		conf.RedisURL = "redis://127.0.0.1:6379/"
		app, _ := NewApp(conf)
		defer app.Close()

		c := app.redis.Get()
		defer c.Close()

		c.Do("SET", "hello", "World")
		world, _ := redis.String(c.Do("GET", "hello"))
		So(world, ShouldEqual, "World")
	})
}
