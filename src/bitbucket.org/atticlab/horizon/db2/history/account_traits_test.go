package history

import (
	"bitbucket.org/atticlab/horizon/test"
	"database/sql"
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
		account := accounts[0]
		// create
		err = q.InsertAccountTraits(AccountTraits{
			TotalOrderID:           account.TotalOrderID,
			AccountAddress:         account.Address,
			BlockOutcomingPayments: true,
		})
		So(err, ShouldBeNil)
		var storedTrait AccountTraits
		storedTrait, err = q.AccountTraitsQ().ForAccount(account.Address)
		So(err, ShouldBeNil)
		So(account.Address, ShouldEqual, storedTrait.AccountAddress)
		So(account.ID, ShouldEqual, storedTrait.ID)
		storedTrait, err = q.AccountTraitsQ().ByID(account.ID)
		So(err, ShouldBeNil)
		So(account.Address, ShouldEqual, storedTrait.AccountAddress)
		So(account.ID, ShouldEqual, storedTrait.ID)
		err = q.DeleteAccountTraits(account.ID)
		So(err, ShouldBeNil)
		_, err = q.AccountTraitsQ().ForAccount(account.Address)
		So(err, ShouldEqual, sql.ErrNoRows)

	})
}
