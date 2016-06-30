package horizon

import (
	"bitbucket.org/atticlab/go-smart-base/amount"
	"bitbucket.org/atticlab/go-smart-base/keypair"
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/assets"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/log"
	"bitbucket.org/atticlab/horizon/render/problem"
	"bitbucket.org/atticlab/horizon/resource"
	"bitbucket.org/atticlab/horizon/resource/base"
	"bitbucket.org/atticlab/horizon/test"
	"encoding/json"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
)

func TestActionsSetCommission(t *testing.T) {
	app := NewTestApp()
	defer app.Close()
	rh := NewRequestHelper(app)
	log.DefaultLogger.Entry.Logger.Level = log.DebugLevel

	Convey("Set commission Actions:", t, func() {
		Convey("Check signature", func() {
			app.unsafeMode = false
			form := url.Values{
				"id": []string{"1"},
			}
			w := rh.Post("/commission", form, test.RequestHelperNoop)
			So(w.Code, ShouldEqual, 401)
		})
		app.unsafeMode = true
		Convey("Invalid id", func() {
			form := url.Values{
				"id": []string{"1"},
			}
			w := rh.Post("/commission", form, test.RequestHelperNoop)
			So(w.Code, ShouldEqual, 404)
		})
		Convey("Invalid asset", func() {
			asset := base.Asset{
				Type:   assets.MustString(xdr.AssetTypeAssetTypeCreditAlphanum4),
				Code:   "EUR",
				Issuer: "random_issuer",
			}
			form := url.Values{
				"asset_type":   []string{asset.Type},
				"asset_code":   []string{asset.Code},
				"asset_issuer": []string{asset.Issuer},
			}
			w := rh.Post("/commission", form, test.RequestHelperNoop)
			checkMalformed(t, w, "asset_issuer")
		})
		Convey("Invalid from", func() {
			form := url.Values{
				"from":   []string{"random_str"},
			}
			w := rh.Post("/commission", form, test.RequestHelperNoop)
			checkMalformed(t, w, "from")
		})
		Convey("Invalid to", func() {
			form := url.Values{
				"to":   []string{"random_str"},
			}
			w := rh.Post("/commission", form, test.RequestHelperNoop)
			checkMalformed(t, w, "to")
		})
		Convey("Invalid from accountType", func() {
			form := url.Values{
				"from_type":   []string{"10"},
			}
			w := rh.Post("/commission", form, test.RequestHelperNoop)
			checkMalformed(t, w, "from_type")
		})
		Convey("Invalid to accountType", func() {
			form := url.Values{
				"to_type":   []string{"10"},
			}
			w := rh.Post("/commission", form, test.RequestHelperNoop)
			checkMalformed(t, w, "to_type")
		})
		Convey("Invalid flat_fee", func() {
			form := url.Values{
				"flat_fee":   []string{"-10"},
			}
			w := rh.Post("/commission", form, test.RequestHelperNoop)
			checkMalformed(t, w, "flat_fee")
		})
		Convey("Invalid percent_fee", func() {
			form := url.Values{
				"percent_fee":   []string{"-10"},
			}
			w := rh.Post("/commission", form, test.RequestHelperNoop)
			checkMalformed(t, w, "percent_fee")
		})
		Convey("valid insert", func() {
			fromKey, err := keypair.Random()
			assert.Nil(t, err)
			from := fromKey.Address()
			toKey, err := keypair.Random()
			assert.Nil(t, err)
			to := toKey.Address()
			fromType := strconv.Itoa(int(xdr.AccountTypeAccountBank))
			toType := strconv.Itoa(int(xdr.AccountTypeAccountDistributionAgent))
			issuer, err := keypair.Random()
			assert.Nil(t, err)
			asset := base.Asset{
				Type:   assets.MustString(xdr.AssetTypeAssetTypeCreditAlphanum4),
				Code:   "EUR",
				Issuer: issuer.Address(),
			}
			flatFee := xdr.Int64(12000000)
			percentFee := xdr.Int64(11)
			form := url.Values{
				"from":         []string{from},
				"to":           []string{to},
				"from_type":    []string{fromType},
				"to_type":      []string{toType},
				"asset_type":   []string{asset.Type},
				"asset_code":   []string{asset.Code},
				"asset_issuer": []string{asset.Issuer},
				"flat_fee":     []string{strconv.FormatInt(int64(flatFee), 10)},
				"percent_fee":  []string{strconv.FormatInt(int64(percentFee), 10)},
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
			Convey("delete", func() {
				deleteForm := url.Values{
					"id": []string{strconv.FormatInt(id, 10)},
					"delete": []string{"true"},
				}
				deleteW := rh.Post("/commission", deleteForm, test.RequestHelperNoop)
				So(deleteW.Code, ShouldEqual, 200)
				var sts []history.Commission
				err = app.historyQ.Commissions().ForAccount(from).Select(&sts)
				assert.Nil(t, err)
				assert.Equal(t, 0, len(sts))
			})
			app.historyQ.DeleteCommissions()
		})

	})
}

func checkMalformed(t * testing.T, w *httptest.ResponseRecorder, fieldName string) {
	So(w.Code, ShouldEqual, 400)
	var p problem.P
	err := json.Unmarshal(w.Body.Bytes(), &p)
	assert.Nil(t, err)
	val, ok := p.Extras["invalid_field"]
	assert.True(t, ok)
	assert.Equal(t, fieldName, val)
}
