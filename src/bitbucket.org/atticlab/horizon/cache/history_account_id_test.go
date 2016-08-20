package cache

import (
	"bitbucket.org/atticlab/horizon/test"
	"testing"
	"bitbucket.org/atticlab/horizon/src/bitbucket.org/atticlab/horizon/db2/history"
)

func TestHistoryAccountID(t *testing.T) {
	tt := test.Start(t).Scenario("base")
	defer tt.Finish()
	db := tt.HorizonRepo()
	c := NewHistoryAccount(&history.Q{
		Repo: db,
	})
	tt.Assert.Equal(0, c.cached.Len())

	id, err := c.Get("GAJLXJ6AJBYG5IDQZQ45CTDYHJRZ6DI4H4IRJA6CD3W6IIJIKLPAS33R")
	if tt.Assert.NoError(err) {
		tt.Assert.Equal(int64(1), id)
		tt.Assert.Equal(1, c.cached.Len())
	}

	id, err = c.Get("NOT_REAL")
	tt.Assert.True(db.NoRows(err))
	tt.Assert.Equal(int64(0), id)
}
