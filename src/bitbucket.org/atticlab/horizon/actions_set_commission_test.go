package horizon

import (
	"testing"
	. "github.com/smartystreets/goconvey/convey"
	"bitbucket.org/atticlab/horizon/test"
	"net/url"
	"bitbucket.org/atticlab/horizon/log"
	"bitbucket.org/atticlab/horizon/resource"
	"bitbucket.org/atticlab/horizon/resource/base"
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"github.com/stretchr/testify/assert"
	"strconv"
	"bitbucket.org/atticlab/go-smart-base/amount"
	"bitbucket.org/atticlab/horizon/assets"
	"bitbucket.org/atticlab/go-smart-base/keypair"
	"bitbucket.org/atticlab/horizon/db2/history"
	"net/http/httptest"
)

func TestActionsSetCommission(t *testing.T) {
	app := NewTestApp()
	defer app.Close()
	rh := NewRequestHelper(app)
	log.DefaultLogger.Entry.Logger.Level = log.DebugLevel

	Convey("Set commission Actions:", t, func() {

		Convey("Invalid id", func() {
			form := url.Values{
				"id": []string{"1"},
			}
			w := rh.Post("/commission", form, test.RequestHelperNoop)
			So(w.Code, ShouldEqual, 404)
		})
		Convey("Invalid asset", func() {
			asset := base.Asset{
				Type: assets.MustString(xdr.AssetTypeAssetTypeCreditAlphanum4),
				Code: "EUR",
				Issuer: "random_issuer",
			}
			form := url.Values{
				"asset_type": []string{asset.Type},
				"asset_code": []string{asset.Code},
				"asset_issuer": []string{asset.Issuer},
			}
			w := rh.Post("/commission", form, test.RequestHelperNoop)
			So(w.Code, ShouldEqual, 400)
		})
		Convey("valid insert", func() {
			from := "random_account"
			to := from + "2"
			fromType := strconv.Itoa(int(xdr.AccountTypeAccountBank))
			toType := strconv.Itoa(int(xdr.AccountTypeAccountDistributionAgent))
			issuer, err := keypair.Random()
			assert.Nil(t, err)
			asset := base.Asset{
				Type: assets.MustString(xdr.AssetTypeAssetTypeCreditAlphanum4),
				Code: "EUR",
				Issuer: issuer.Address(),
			}
			flatFee := xdr.Int64(12000000)
			percentFee := xdr.Int64(11)
			form := url.Values{
				"from": []string{from},
				"to": []string{to},
				"from_type": []string{fromType},
				"to_type": []string{toType},
				"asset_type": []string{asset.Type},
				"asset_code": []string{asset.Code},
				"asset_issuer": []string{asset.Issuer},
				"flat_fee": []string{strconv.FormatInt(int64(flatFee), 10)},
				"percent_fee": []string{strconv.FormatInt(int64(percentFee), 10)},
			}
			w := rh.Post("/commission", form, test.RequestHelperNoop)
			log.WithField("Response", w.Body.String()).Debug("Got response")
			check := func(w *httptest.ResponseRecorder) int64 {
				So(w.Code, ShouldEqual, 200)
				var sts []history.Commission
				err = app.historyQ.Commissions().ForAccount(from).Select(&sts)
				assert.Nil(t, err)
				assert.Equal(t, 1, len(sts))
				st := resource.Commission{}
				st.Populate(sts[0])
				assert.Equal(t, from, *st.From)
				assert.Equal(t, to, *st.To)
				assert.Equal(t, fromType, strconv.Itoa(int(*st.FromAccountTypeI)))
				assert.Equal(t, toType, strconv.Itoa(int(*st.ToAccountTypeI)))
				assert.Equal(t, asset, *st.Asset)
				assert.Equal(t, amount.String(flatFee), st.FlatFee)
				assert.Equal(t, amount.String(percentFee), st.PercentFee)
				return st.Id
			}
			id := check(w)
			Convey("update", func() {
				flatFee = 99
				form.Set("id", strconv.FormatInt(id, 10))
				form.Set("flat_fee", strconv.FormatInt(int64(flatFee), 10))
				updateW := rh.Post("/commission", form, test.RequestHelperNoop)
				check(updateW)
			})
			app.historyQ.DeleteCommissions()
		})

	})
}
