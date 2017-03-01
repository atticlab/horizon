package redis

import (
	"bitbucket.org/atticlab/horizon/log"
	"github.com/stretchr/testify/mock"
	"time"
)

type ConnectionProviderMock struct {
	mock.Mock
}

func (p *ConnectionProviderMock) GetConnection() ConnectionInterface {
	a := p.Called()
	return a.Get(0).(ConnectionInterface)
}

type ConnectionMock struct {
	mock.Mock
}

func (m *ConnectionMock) HMSet(args ...interface{}) error {
	log.Panic("Not implemented")
	return nil
}

func (m *ConnectionMock) HGetAll(key string) (interface{}, error) {
	log.Panic("Not implemented")
	return nil, nil
}

func (m *ConnectionMock) Expire(key string, timeout time.Duration) (bool, error) {
	log.Panic("Not implemented")
	return false, nil
}

func (m *ConnectionMock) GetSet(key string, data interface{}) (interface{}, error) {
	log.Panic("Not implemented")
	return nil, nil
}

func (m *ConnectionMock) Get(key string) (interface{}, error) {
	log.Panic("Not implemented")
	return nil, nil
}

func (m *ConnectionMock) Set(key string, data interface{}) error {
	log.Panic("Not implemented")
	return nil
}

func (m *ConnectionMock) Watch(key string) error {
	a := m.Called(key)
	return a.Error(0)
}

func (m *ConnectionMock) UnWatch() error {
	a := m.Called()
	return a.Error(0)
}

func (m *ConnectionMock) Multi() error {
	a := m.Called()
	return a.Error(0)
}

func (m *ConnectionMock) Exec() (bool, error) {
	a := m.Called()
	return a.Get(0).(bool), a.Error(1)
}

func (m *ConnectionMock) Close() error {
	a := m.Called()
	return a.Error(0)
}

func (m *ConnectionMock) Delete(key string) error {
	return m.Called(key).Error(0)
}

func (m *ConnectionMock) Ping() error {
	return nil
}
