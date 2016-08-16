package history

import (
	"bitbucket.org/atticlab/horizon/test"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestAccountTraitsQueries(t *testing.T) {
	tt := test.Start(t).Scenario("base")
	defer tt.Finish()
	q := &Q{tt.HorizonRepo()}

	Convey("AccountTraits", t, func() {
		var accounts []Account
		err := q.Accounts().Select(&accounts)
		So(err, ShouldBeNil)
		So(len(accounts), ShouldBeGreaterThanOrEqualTo, 2)
		accounts = accounts[0:2]
		Convey("Select for account", func() {
			var storedTrait AccountTraits
			storedTrait, err = q.AccountTraitsQ().ForAccount(accounts[0].Address)
			So(err, ShouldBeNil)
			So(accounts[0].Address, ShouldEqual, storedTrait.AccountAddress)
			So(accounts[0].ID, ShouldEqual, storedTrait.ID)
		})
		Convey("Select by id", func() {
			var storedTrait AccountTraits
			storedTrait, err = q.AccountTraitsQ().ByID(accounts[0].ID)
			So(err, ShouldBeNil)
			So(accounts[0].Address, ShouldEqual, storedTrait.AccountAddress)
			So(accounts[0].ID, ShouldEqual, storedTrait.ID)
		})

	})
}
