package test

import (
	"bitbucket.org/atticlab/horizon/db2/core"
	"github.com/stretchr/testify/mock"
)

type SignersProviderMock struct {
	mock.Mock
}

func (m *SignersProviderMock) SignersByAddress(dest interface{}, addy string) error {
	a := m.Called(addy)
	signers := a.Get(0).([]core.Signer)
	destSigners := dest.(*[]core.Signer)
	*destSigners = signers
	return a.Error(1)
}
