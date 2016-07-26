package txsub

import (
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"github.com/stretchr/testify/mock"
)

type TransactionValidatorMock struct {
	mock.Mock
}

func (v *TransactionValidatorMock) CheckTransaction(tx *xdr.TransactionEnvelope) error {
	a := v.Called(tx)
	return a.Error(0)
}
