package cache

import (
	"bitbucket.org/atticlab/horizon/test"
	"testing"
	"bitbucket.org/atticlab/horizon/db2/history"
)

func TestHistoryAccountID(t *testing.T) {
	tt := test.Start(t).Scenario("base")
	defer tt.Finish()
	db := tt.HorizonRepo()
	c := NewHistoryAccount(&history.Q{
		Repo: db,
	})
	tt.Assert.Equal(0, c.Cache.ItemCount())

	address := "GAJLXJ6AJBYG5IDQZQ45CTDYHJRZ6DI4H4IRJA6CD3W6IIJIKLPAS33R"
	account, err := c.Get(address)
	if tt.Assert.NoError(err) {
		tt.Assert.Equal(&history.Account{
			TotalOrderID: history.TotalOrderID{
				ID: 1,
			},
			Address: address,
		}, account)
		tt.Assert.Equal(1, c.Cache.ItemCount())
	}

	account, err = c.Get("NOT_REAL")
	tt.Assert.True(db.NoRows(err))
	var noAccount *history.Account
	tt.Assert.Equal(noAccount, account)
}
