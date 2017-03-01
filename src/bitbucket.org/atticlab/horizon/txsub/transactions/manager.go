package transactions

import (
	"bitbucket.org/atticlab/horizon/cache"
	"bitbucket.org/atticlab/horizon/config"
	"bitbucket.org/atticlab/horizon/db2/core"
	"bitbucket.org/atticlab/horizon/db2/history"
)

type Manager struct {
	*cache.SharedCache
	CoreQ        core.QInterface
	HistoryQ     history.QInterface
	Config       *config.Config
}

func NewManager(core core.QInterface, history history.QInterface, config *config.Config, sharedCache *cache.SharedCache) *Manager {
	return &Manager{
		CoreQ:        core,
		HistoryQ:     history,
		Config:       config,
		SharedCache:  sharedCache,
	}
}
