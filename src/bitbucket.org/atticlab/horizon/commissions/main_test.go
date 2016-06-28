package commissions

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"github.com/stretchr/testify/assert"
	"math"
	"bitbucket.org/atticlab/go-smart-base/amount"
)

func TestCommission(t *testing.T) {
	Convey("countPercentFee", t, func() {
		percentFee := xdr.Int64(1 * amount.One) // fee is 1%
		Convey("amount too small", func() {
			fee := countPercentFee(xdr.Int64(1), percentFee)
			assert.Equal(t, xdr.Int64(0), fee)
		})
		Convey("amount is ok", func() {
			paymentAmount := 1230 * amount.One
			fee := countPercentFee(xdr.Int64(paymentAmount), percentFee)
			assert.Equal(t, xdr.Int64(12.3 * amount.One), fee)
		})
		Convey("fee cutted", func() {
			paymentAmount := 156
			fee := countPercentFee(xdr.Int64(paymentAmount), percentFee)
			assert.Equal(t, xdr.Int64(1), fee)
		})
		Convey("fee cutted not rounded", func() {
			paymentAmount := 1560
			fee := countPercentFee(xdr.Int64(paymentAmount), percentFee)
			assert.Equal(t, xdr.Int64(15), fee)
		})
		Convey("amount is big", func () {
			paymentAmount := math.MaxInt64
			fee := countPercentFee(xdr.Int64(paymentAmount), percentFee)
			assert.Equal(t, xdr.Int64(paymentAmount/100), fee)
		})
	})
}
