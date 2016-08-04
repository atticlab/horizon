package redis

import "bitbucket.org/atticlab/horizon/log"

type ConnectionProviderInterface interface {
	GetConnection() ConnectionInterface
}

type ConnectionProvider struct {

}

func NewConnectionProvider() ConnectionProviderInterface {
	return &ConnectionProvider{}
}

func (c ConnectionProvider) GetConnection() ConnectionInterface {
	if redisPool == nil {
		log.Panic("Redis must be initialized")
	}
	return NewConnection(redisPool.Get())
}
