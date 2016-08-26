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
	"math/rand"
)

func TestTraits(t *testing.T) {
	histMock := history.QMock{}
	traitsMock := history.AccountTraitsQMock{}
	histMock.On("AccountTraitsQ").Return(&traitsMock)
	traits := NewTraitsValidator(&histMock)
	Convey("Traits test:", t, func() {
		sourceKP, err := keypair.Random()
		So(err, ShouldBeNil)
		source := &history.Account{
			TotalOrderID: history.TotalOrderID{
				ID: rand.Int63(),
			},
			Address: sourceKP.Address(),
		}
		destKP, err := keypair.Random()
		So(err, ShouldBeNil)
		dest := &history.Account{
			TotalOrderID: history.TotalOrderID{
				ID: rand.Int63(),
			},
			Address: destKP.Address(),
		}
		Convey("Both accounts does not have traits", func() {
			traitsMock.On("ByID", source.ID).Return(history.AccountTraits{}, sql.ErrNoRows).Once()
			traitsMock.On("ByID", dest.ID).Return(history.AccountTraits{}, sql.ErrNoRows).Once()
			result, err := traits.CheckTraits(source, dest)
			So(err, ShouldBeNil)
			So(result, ShouldBeNil)
		})
		Convey("Both accounts have traits, but not blocked", func() {
			traitsMock.On("ByID", source.ID).Return(history.AccountTraits{
				BlockOutcomingPayments: false,
				BlockIncomingPayments:  true,
			}, nil).Once()
			traitsMock.On("ByID", dest.ID).Return(history.AccountTraits{
				BlockOutcomingPayments: true,
				BlockIncomingPayments:  false,
			}, nil).Once()
			result, err := traits.CheckTraits(source, dest)
			So(err, ShouldBeNil)
			So(result, ShouldBeNil)
		})
		Convey("Source is blocked", func() {
			traitsMock.On("ByID", source.ID).Return(history.AccountTraits{
				BlockOutcomingPayments: true,
				BlockIncomingPayments:  false,
			}, nil).Once()
			result, err := traits.CheckTraits(source, dest)
			So(err, ShouldBeNil)
			assert.Equal(t, result, &results.RestrictedForAccountError{
				Reason: fmt.Sprintf("Outcoming payments for account (%s) are restricted by administrator.", source.Address),
			})
		})
		Convey("Dest is blocked", func() {
			traitsMock.On("ByID", source.ID).Return(history.AccountTraits{
				BlockOutcomingPayments: false,
				BlockIncomingPayments:  true,
			}, nil).Once()
			traitsMock.On("ByID", dest.ID).Return(history.AccountTraits{
				BlockOutcomingPayments: false,
				BlockIncomingPayments:  true,
			}, nil).Once()
			result, err := traits.CheckTraits(source, dest)
			So(err, ShouldBeNil)
			assert.Equal(t, result, &results.RestrictedForAccountError{
				Reason: fmt.Sprintf("Incoming payments for account (%s) are restricted by administrator.", dest.Address),
			})
		})

	})
}
