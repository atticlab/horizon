package validators

import (
	"testing"

	"bitbucket.org/atticlab/go-smart-base/keypair"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/txsub/results"
	"database/sql"
	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
)

func TestTraits(t *testing.T) {
	histMock := history.QMock{}
	traits := NewTraitsValidator(&histMock)
	Convey("Traits test:", t, func() {
		source, err := keypair.Random()
		So(err, ShouldBeNil)
		dest, err := keypair.Random()
		So(err, ShouldBeNil)
		Convey("Both accounts does not have traits", func() {
			histMock.On("GetAccountTraitsByAddress", source.Address()).Return(nil, sql.ErrNoRows)
			histMock.On("GetAccountTraitsByAddress", dest.Address()).Return(nil, sql.ErrNoRows)
			result, err := traits.CheckTraits(source.Address(), dest.Address())
			So(err, ShouldBeNil)
			So(result, ShouldBeNil)
		})
		Convey("Both accounts have traits, but not blocked", func() {
			histMock.On("GetAccountTraitsByAddress", source.Address()).Return(history.AccountTraits{
				BlockIncomingPayments:  true,
				BlockOutcomingPayments: false,
			}, nil)
			histMock.On("GetAccountTraitsByAddress", dest.Address()).Return(history.AccountTraits{
				BlockIncomingPayments:  false,
				BlockOutcomingPayments: true,
			}, nil)
			result, err := traits.CheckTraits(source.Address(), dest.Address())
			So(err, ShouldBeNil)
			So(result, ShouldBeNil)
		})
		Convey("Source is blocked", func() {
			histMock.On("GetAccountTraitsByAddress", source.Address()).Return(history.AccountTraits{
				BlockOutcomingPayments: true,
			}, nil)
			result, err := traits.CheckTraits(source.Address(), dest.Address())
			So(err, ShouldBeNil)
			assert.Equal(t, result, &results.RestrictedForAccountError{
				Reason: fmt.Sprintf("Outcoming payments for account (%s) are restricted by administrator.", source.Address()),
			})
		})
		Convey("Dest is blocked", func() {
			histMock.On("GetAccountTraitsByAddress", source.Address()).Return(history.AccountTraits{
				BlockOutcomingPayments: false,
			}, nil)
			histMock.On("GetAccountTraitsByAddress", dest.Address()).Return(history.AccountTraits{
				BlockIncomingPayments: true,
			}, nil)
			result, err := traits.CheckTraits(source.Address(), dest.Address())
			So(err, ShouldBeNil)
			assert.Equal(t, result, &results.RestrictedForAccountError{
				Reason: fmt.Sprintf("Incoming payments for account (%s) are restricted by administrator.", dest.Address()),
			})
		})

	})
}
