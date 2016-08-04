package validators

import (
	"testing"

	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/db2/core"
	. "github.com/smartystreets/goconvey/convey"
)

func TestAccountTypes(t *testing.T) {
	Convey("VerifyAccountTypesForPayment:", t, func() {
		Convey("Bank can't send to anon user", func() {
			source := core.Account{
				AccountType: xdr.AccountTypeAccountBank,
			}
			dest := core.Account{
				AccountType: xdr.AccountTypeAccountAnonymousUser,
			}
			validator := NewAccountTypeValidator()
			err := validator.VerifyAccountTypesForPayment(source, dest)
			So(err, ShouldNotBeNil)
		})

	})
}
