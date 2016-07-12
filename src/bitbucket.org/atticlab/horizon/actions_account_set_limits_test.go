package horizon

import (
	"bitbucket.org/atticlab/go-smart-base/keypair"
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/db2/core"
	coreTest "bitbucket.org/atticlab/horizon/db2/core/test"
	"bitbucket.org/atticlab/horizon/render/problem"
	"bitbucket.org/atticlab/horizon/resource"
	"bitbucket.org/atticlab/horizon/test"
	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"net/url"
	"testing"
	"encoding/json"
)

func TestActionsSetLimits(t *testing.T) {
	test.LoadScenario("base")
	app := NewTestApp()
	defer app.Close()
	rh := NewRequestHelper(app)
	signersProviderMock := coreTest.SignersProviderMock{}
	app.SetSignersProvider(&signersProviderMock)
	signer, err := keypair.Random()
	assert.Nil(t, err)
	account := app.config.BankMasterKey
	path := fmt.Sprintf("/accounts/%s/limits", account)
	Convey("Check signature", t, func() {
		form := url.Values{}
		w := rh.Post(path, form, test.RequestHelperNoop)
		So(w.Code, ShouldEqual, 401)
	})
	Convey("Set limits", t, func() {
		signersProviderMock.On("SignersByAddress", app.config.BankMasterKey).Return([]core.Signer{core.Signer{
			Accountid:  "1",
			Publickey:  signer.Address(),
			Weight:     1,
			SignerType: uint32(xdr.SignerTypeSignerAdmin),
		}}, nil)
		Convey("Invalid account", func() {
			w := rh.SignedPost(signer, "/accounts/invalid_account/limits", url.Values{}, test.RequestHelperNoop)
			So(w.Code, ShouldEqual, 400)
			So(w.Body, ShouldBeProblem, problem.BadRequest, "account_id")
		})
		Convey("account does not exist", func() {
			newAccount, err := keypair.Random()
			assert.Nil(t, err)
			newAccountPath := fmt.Sprintf("/accounts/%s/limits", newAccount.Address())
			w := rh.SignedPost(signer, newAccountPath, url.Values{}, test.RequestHelperNoop)
			So(w.Code, ShouldEqual, 404)
			So(w.Body, ShouldBeProblem, problem.NotFound)
		})
		Convey("happy path", func() {
			// create new limit
			var actual resource.AccountLimits
			expected := resource.AccountLimitsEntry{
				AssetCode: "USD",
				MaxOperationOut: 1,
				DailyMaxOut: 2,
				MonthlyMaxOut: 3,
				MaxOperationIn: 5,
				DailyMaxIn: 7,
				MonthlyMaxIn: 11,
			}
			w := rh.SignedPost(signer, path, url.Values{
				"asset_code": []string{"USD"},
				"max_operation_out": []string{"1"},
				"daily_max_out": []string{"2"},
				"monthly_max_out": []string{"3"},
				"max_operation_in": []string{"5"},
				"daily_max_in": []string{"7"},
				"monthly_max_in": []string{"11"},
			}, test.RequestHelperNoop)
			So(w.Code, ShouldEqual, 200)
			err := json.Unmarshal(w.Body.Bytes(), &actual)
			assert.Nil(t, err)
			assert.Len(t, actual.Limits, 1)
			assert.Equal(t, expected, actual.Limits[0])
			// update
			expected.DailyMaxIn = 13
			expected.MaxOperationOut = 17
			w = rh.SignedPost(signer, path, url.Values{
				"asset_code": []string{"USD"},
				"max_operation_out": []string{"17"},
				"daily_max_out": []string{"2"},
				"monthly_max_out": []string{"3"},
				"max_operation_in": []string{"5"},
				"daily_max_in": []string{"13"},
				"monthly_max_in": []string{"11"},
			}, test.RequestHelperNoop)
			So(w.Code, ShouldEqual, 200)
			err = json.Unmarshal(w.Body.Bytes(), &actual)
			assert.Nil(t, err)
			assert.Len(t, actual.Limits, 1)
			assert.Equal(t, expected, actual.Limits[0])
		})
	})
}
