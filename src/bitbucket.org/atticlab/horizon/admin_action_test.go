package horizon

import (
	"bitbucket.org/atticlab/go-smart-base/keypair"
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/db2/core"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/log"
	"bitbucket.org/atticlab/horizon/render/problem"
	"bitbucket.org/atticlab/horizon/test"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	coreTest "bitbucket.org/atticlab/horizon/db2/core/test"
	"net/url"
	"testing"
	"time"
	"encoding/json"
)

func TestAdminAction(t *testing.T) {
	app := NewTestApp()
	defer app.Close()

	log.DefaultLogger.Entry.Logger.Level = log.DebugLevel

	err := app.HistoryQ().DeleteAuditLog()
	assert.Nil(t, err)

	Convey("Admin action", t, func() {
		signer, err := keypair.Random()
		assert.Nil(t, err)

		signersProviderMock := coreTest.SignersProviderMock{}

		action := AdminAction{}
		action.App = app

		bodyData := url.Values{
			"data": []string{"random_data"},
		}
		requestData := test.NewRequestData(signer, bodyData)
		action.signersProvider = &signersProviderMock
		coreSigner := core.Signer{
			Accountid:  "1",
			Publickey:  signer.Address(),
			Weight:     1,
			SignerType: uint32(xdr.SignerTypeSignerAdmin),
		}
		action.Info.ActionPerformed = "testring"
		action.Info.Subject = "admin_action"
		Convey("Valid request", func() {
			err := app.HistoryQ().DeleteAuditLog()
			assert.Nil(t, err)
			err = app.HistoryQ().DeleteCommissions()
			assert.Nil(t, err)
			signersProviderMock.On("SignersByAddress", app.config.BankMasterKey).Return([]core.Signer{coreSigner}, nil)
			action.R = requestData.CreateRequest()
			action.StartAdminAction()
			// make sure we can word with db
			key := history.CommissionKey{}
			hash, _ := key.Hash()
			commission := history.Commission{
				KeyHash:    hash,
				KeyValue:   "{}",
				FlatFee:    int64(1000),
				PercentFee: int64(2000),
			}
			meta := commission
			action.Info.Meta = &meta
			app.HistoryQ().InsertCommission(&commission)
			action.FinishAdminAction()
			So(action.Err, ShouldBeNil)
			var comms []history.Commission
			err = app.HistoryQ().Commissions().Select(&comms)
			assert.Nil(t, err)
			assert.Equal(t, 1, len(comms))
			commission.Id = comms[0].Id
			assert.Equal(t, commission, comms[0])
			logs, err := app.HistoryQ().GetAllAuditLogs()
			assert.Nil(t, err)
			assert.Equal(t, 1, len(logs))
			assert.Equal(t, action.Info.ActorPublicKey.Address(), logs[0].Actor)
			assert.Equal(t, string(action.Info.Subject), logs[0].Subject)
			assert.Equal(t, string(action.Info.ActionPerformed), logs[0].Action)
			var storedMeta history.Commission
			err = json.Unmarshal([]byte(logs[0].Meta), &storedMeta)
			assert.Nil(t, err)
			assert.Equal(t, action.Info.Meta, &storedMeta)
			err = app.HistoryQ().DeleteAuditLog()
			assert.Nil(t, err)
			err = app.HistoryQ().DeleteCommissions()
			assert.Nil(t, err)
		})
		Convey("Failed during admin action", func() {
			err := app.HistoryQ().DeleteAuditLog()
			assert.Nil(t, err)
			err = app.HistoryQ().DeleteCommissions()
			assert.Nil(t, err)
			signersProviderMock.On("SignersByAddress", app.config.BankMasterKey).Return([]core.Signer{coreSigner}, nil)
			action.R = requestData.CreateRequest()
			log.Debug("Starting admin action")
			action.StartAdminAction()
			key := history.CommissionKey{}
			hash, _ := key.Hash()
			commission := history.Commission{
				KeyHash:    hash,
				KeyValue:   "{}",
				FlatFee:    int64(1000),
				PercentFee: int64(2000),
			}
			err = action.HistoryQ().InsertCommission(&commission)
			assert.Nil(t, err)
			action.Err = &problem.BadRequest
			action.FinishAdminAction()
			So(action.Err, ShouldNotBeNil)
			var comms []history.Commission
			err = action.HistoryQ().Commissions().Select(&comms)
			assert.Nil(t, err)
			assert.Equal(t, 0, len(comms))
			logs, err := app.HistoryQ().GetAllAuditLogs()
			assert.Nil(t, err)
			assert.Equal(t, 0, len(logs))
			err = app.HistoryQ().DeleteAuditLog()
			assert.Nil(t, err)
			err = app.HistoryQ().DeleteCommissions()
			assert.Nil(t, err)
		})
		Convey("Invalid signature", func() {
			requestData = test.NewRequestData(signer, bodyData)
			Convey("Signature not set", func() {
				requestData.Signature = ""
				action.R = requestData.CreateRequest()
				action.StartAdminAction()
				action.FinishAdminAction()
				So(action.Err, ShouldNotBeNil)
				So(action.Err, ShouldEqual, &Unauthorized)
			})
			Convey("Signer not set", func() {
				requestData.PublicKey = ""
				action.R = requestData.CreateRequest()
				action.StartAdminAction()
				action.FinishAdminAction()
				So(action.Err, ShouldNotBeNil)
				So(action.Err, ShouldEqual, &Unauthorized)
			})
			Convey("Signature expired", func() {
				requestData.Timestamp = requestData.Timestamp - int64(time.Duration(app.config.AdminSignatureValid)*time.Second*2)
				action.R = requestData.CreateRequest()
				action.StartAdminAction()
				action.FinishAdminAction()
				So(action.Err, ShouldNotBeNil)
				So(action.Err, ShouldEqual, &Unauthorized)
			})
			Convey("Signature does not match content", func() {
				bodyData.Add("new_random_key", "new_random_value")
				requestData.EncodedForm = bodyData.Encode()
				action.R = requestData.CreateRequest()
				action.StartAdminAction()
				action.FinishAdminAction()
				So(action.Err, ShouldNotBeNil)
				So(action.Err, ShouldEqual, &Unauthorized)
			})
			Convey("Signer is not admin", func() {
				newSigner, err := keypair.Random()
				So(err, ShouldBeNil)
				signersProviderMock.On("SignersByAddress", app.config.BankMasterKey).Return([]core.Signer{core.Signer{
					Accountid:  "2",
					Publickey:  newSigner.Address(),
					Weight:     1,
					SignerType: uint32(xdr.SignerTypeSignerGeneral),
				}}, nil)
				newRequestData := test.NewRequestData(newSigner, bodyData)
				action.R = newRequestData.CreateRequest()
				action.StartAdminAction()
				action.FinishAdminAction()
				So(action.Err, ShouldNotBeNil)
				So(action.Err, ShouldEqual, &Unauthorized)
			})

		})
		err = app.HistoryQ().DeleteAuditLog()
		assert.Nil(t, err)
	})
}
