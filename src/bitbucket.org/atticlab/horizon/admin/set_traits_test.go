package admin

import (
	"bitbucket.org/atticlab/go-smart-base/keypair"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/log"
	"bitbucket.org/atticlab/horizon/test"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestActionsSetTraits(t *testing.T) {
	tt := test.Start(t).Scenario("base")
	defer tt.Finish()

	log.DefaultLogger.Entry.Logger.Level = log.DebugLevel
	historyQ := &history.Q{tt.HorizonRepo()}
	account := test.NewTestConfig().BankMasterKey

	Convey("Set traits", t, func() {
		Convey("Invalid account", func() {
			action := NewSetTraitsAction(NewAdminAction(map[string]interface{}{
				"account_id": "invalid_id",
			}, historyQ))
			action.Validate()
			So(action.Err, ShouldNotBeNil)
			So(action.Err, ShouldBeInvalidField, "account_id")
		})
		Convey("Invalid block_incoming_payments", func() {
			action := NewSetTraitsAction(NewAdminAction(map[string]interface{}{
				"account_id":              account,
				"block_incoming_payments": "not_bool",
			}, historyQ))
			action.Validate()
			So(action.Err, ShouldNotBeNil)
			So(action.Err, ShouldBeInvalidField, "block_incoming_payments")
		})
		Convey("Invalid block_outcoming_payments", func() {
			action := NewSetTraitsAction(NewAdminAction(map[string]interface{}{
				"account_id":               account,
				"block_outcoming_payments": "not_bool",
			}, historyQ))
			action.Validate()
			So(action.Err, ShouldNotBeNil)
			So(action.Err, ShouldBeInvalidField, "block_outcoming_payments")
		})
		Convey("account does not exist", func() {
			newAccount, err := keypair.Random()
			assert.Nil(t, err)
			action := NewSetTraitsAction(NewAdminAction(map[string]interface{}{
				"account_id":               newAccount.Address(),
				"block_outcoming_payments": "not_bool",
			}, historyQ))
			action.Validate()
			So(action.Err, ShouldNotBeNil)
			So(action.Err, ShouldBeInvalidField, "block_outcoming_payments")
		})
		Convey("happy path", func() {
			// create new trait
			var storedAcc history.Account
			err := historyQ.AccountByAddress(&storedAcc, account)
			So(err, ShouldBeNil)
			expected := history.AccountTraits{
				TotalOrderID:           storedAcc.TotalOrderID,
				BlockIncomingPayments:  true,
				BlockOutcomingPayments: false,
			}
			action := NewSetTraitsAction(NewAdminAction(map[string]interface{}{
				"account_id":              account,
				"block_incoming_payments": "true",
			}, historyQ))
			checkTraitsAction(action, account, expected, historyQ)
			// update
			expected.BlockOutcomingPayments = true
			action = NewSetTraitsAction(NewAdminAction(map[string]interface{}{
				"account_id":               account,
				"block_outcoming_payments": "true",
			}, historyQ))
			checkTraitsAction(action, account, expected, historyQ)
			// remove
			expected.BlockOutcomingPayments = false
			expected.BlockIncomingPayments = false
			action = NewSetTraitsAction(NewAdminAction(map[string]interface{}{
				"account_id":               account,
				"block_incoming_payments":  "false",
				"block_outcoming_payments": "false",
			}, historyQ))
			checkTraitsAction(action, account, expected, historyQ)
		})
	})
}

func checkTraitsAction(action *SetTraitsAction, account string, expected history.AccountTraits, historyQ *history.Q) {
	action.Validate()
	So(action.Err, ShouldBeNil)
	action.Apply()
	So(action.Err, ShouldBeNil)
	var actual history.AccountTraits
	err := historyQ.GetAccountTraitsByAddress(&actual, account)
	So(err, ShouldBeNil)
	So(actual.TotalOrderID.ID, ShouldEqual, expected.TotalOrderID.ID)
	So(actual.BlockIncomingPayments, ShouldEqual, expected.BlockIncomingPayments)
	So(actual.BlockOutcomingPayments, ShouldEqual, expected.BlockOutcomingPayments)
}
