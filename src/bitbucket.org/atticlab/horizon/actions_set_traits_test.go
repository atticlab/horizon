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

func TestActionsSetTraits(t *testing.T) {
	test.LoadScenario("base")
	app := NewTestApp()
	defer app.Close()
	rh := NewRequestHelper(app)
	signersProviderMock := coreTest.SignersProviderMock{}
	app.SetSignersProvider(&signersProviderMock)
	signer, err := keypair.Random()
	assert.Nil(t, err)
	account := app.config.BankMasterKey
	path := fmt.Sprintf("/accounts/%s/traits", account)
	Convey("Check signature", t, func() {
		form := url.Values{}
		w := rh.Post(path, form, test.RequestHelperNoop)
		So(w.Code, ShouldEqual, 401)
	})
	Convey("Set traits", t, func() {
		signersProviderMock.On("SignersByAddress", app.config.BankMasterKey).Return([]core.Signer{core.Signer{
			Accountid:  "1",
			Publickey:  signer.Address(),
			Weight:     1,
			SignerType: uint32(xdr.SignerTypeSignerAdmin),
		}}, nil)
		Convey("Invalid account", func() {
			w := rh.SignedPost(signer, "/accounts/invalid_account/traits", url.Values{}, test.RequestHelperNoop)
			So(w.Code, ShouldEqual, 400)
			So(w.Body, ShouldBeProblem, problem.BadRequest, "account_id")
		})
		Convey("Invalid block_incoming_payments", func() {
			w := rh.SignedPost(signer, path, url.Values{
				"block_incoming_payments": []string{"not_bool"},
			}, test.RequestHelperNoop)
			So(w.Code, ShouldEqual, 400)
			So(w.Body, ShouldBeProblem, problem.BadRequest, "block_incoming_payments")
		})
		Convey("Invalid block_outcoming_payments", func() {
			w := rh.SignedPost(signer, path, url.Values{
				"block_outcoming_payments": []string{"not_bool"},
			}, test.RequestHelperNoop)
			So(w.Code, ShouldEqual, 400)
			So(w.Body, ShouldBeProblem, problem.BadRequest, "block_outcoming_payments")
		})
		Convey("account does not exist", func() {
			newAccount, err := keypair.Random()
			assert.Nil(t, err)
			newAccountPath := fmt.Sprintf("/accounts/%s/traits", newAccount.Address())
			w := rh.SignedPost(signer, newAccountPath, url.Values{
				"block_outcoming_payments": []string{"true"},
			}, test.RequestHelperNoop)
			So(w.Code, ShouldEqual, 404)
			So(w.Body, ShouldBeProblem, problem.NotFound)
		})
		Convey("happy path", func() {
			// create new trait
			var actual resource.AccountTraits
			expected := resource.AccountTraits{
				BlockIncomingPayments:  true,
				BlockOutcomingPayments: false,
			}
			w := rh.SignedPost(signer, path, url.Values{
				"block_incoming_payments": []string{"true"},
			}, test.RequestHelperNoop)
			So(w.Code, ShouldEqual, 200)
			err := json.Unmarshal(w.Body.Bytes(), &actual)
			assert.Nil(t, err)
			assert.Equal(t, expected.BlockIncomingPayments, actual.BlockIncomingPayments)
			assert.Equal(t, expected.BlockOutcomingPayments, actual.BlockOutcomingPayments)
			// update
			expected = resource.AccountTraits{
				BlockIncomingPayments:  true,
				BlockOutcomingPayments: true,
			}
			w = rh.SignedPost(signer, path, url.Values{
				"block_outcoming_payments": []string{"true"},
			}, test.RequestHelperNoop)
			So(w.Code, ShouldEqual, 200)
			err = json.Unmarshal(w.Body.Bytes(), &actual)
			assert.Nil(t, err)
			assert.Equal(t, expected.BlockIncomingPayments, actual.BlockIncomingPayments)
			assert.Equal(t, expected.BlockOutcomingPayments, actual.BlockOutcomingPayments)
			// remove
			expected = resource.AccountTraits{
				BlockIncomingPayments:  true,
				BlockOutcomingPayments: false,
			}
			w = rh.SignedPost(signer, path, url.Values{
				"block_outcoming_payments": []string{"false"},
			}, test.RequestHelperNoop)
			So(w.Code, ShouldEqual, 200)
			err = json.Unmarshal(w.Body.Bytes(), &actual)
			assert.Nil(t, err)
			assert.Equal(t, expected.BlockIncomingPayments, actual.BlockIncomingPayments)
			assert.Equal(t, expected.BlockOutcomingPayments, actual.BlockOutcomingPayments)
		})
	})
}
