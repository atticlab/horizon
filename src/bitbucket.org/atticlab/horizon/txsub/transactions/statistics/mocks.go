package statistics

import (
	"github.com/stretchr/testify/mock"
	"time"
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/db2/history"
)

type ManagerMock struct {
	mock.Mock
}

func (m *ManagerMock) UpdateGet(paymentData *PaymentData, paymentDirection PaymentDirection, now time.Time) (result map[xdr.AccountType]history.AccountStatistics, err error) {
	a := m.Called(paymentData, paymentDirection, now)
	return a.Get(0).(map[xdr.AccountType]history.AccountStatistics), a.Error(1)
}

func (m *ManagerMock) CancelOp(paymentData *PaymentData, paymentDirection PaymentDirection, now time.Time) error {
	return m.Called(paymentData, paymentDirection, now).Error(0)
}

